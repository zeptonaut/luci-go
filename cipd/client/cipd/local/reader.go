// Copyright 2014 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package local

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"

	"github.com/luci/luci-go/cipd/client/cipd/common"
	"github.com/luci/luci-go/common/clock"
	"github.com/luci/luci-go/common/logging"
)

// VerificationMode is defines whether to verify hash or not.
type VerificationMode int

const (
	// VerifyHash instructs OpenPackage to calculate hash of the package and
	// compare it to the given instanceID.
	VerifyHash VerificationMode = 0

	// SkipVerification instructs OpenPackage to skip the hash verification and
	// trust that the given instanceID matches the package.
	SkipHashVerification VerificationMode = 1
)

var ErrHashMismatch = errors.New("package hash mismatch")

// PackageInstance represents a binary CIPD package file (with manifest inside).
type PackageInstance interface {
	// Pin identifies package name and concreted instance ID of this package file.
	Pin() common.Pin

	// Files returns a list of files to deploy with the package.
	Files() []File

	// DataReader returns reader that reads raw package data.
	DataReader() io.ReadSeeker
}

// InstanceFile is an underlying data file for a PackageInstance.
type InstanceFile interface {
	io.ReadSeeker

	// Close is a bit non-standard, and can be used to indicate to the storage
	// (filesystem and/or cache) layer that this instance is actaully bad. The
	// storage layer can then evict/revoke, etc. the bad file.
	Close(ctx context.Context, corrupt bool) error
}

// OpenInstance prepares the package for extraction.
//
// If instanceID is an empty string, OpenInstance will calculate the hash
// of the package and use it as InstanceID (regardless of verification mode).
//
// If instanceID is not empty and verification mode is VerifyHash,
// OpenInstance will check that package data matches the given instanceID. It
// skips this check if verification mode is SkipHashVerification.
func OpenInstance(ctx context.Context, r InstanceFile, instanceID string, v VerificationMode) (PackageInstance, error) {
	out := &packageInstance{data: r}
	if err := out.open(instanceID, v); err != nil {
		return nil, err
	}
	return out, nil
}

type dummyInstance struct {
	*os.File
}

func (d dummyInstance) Close(context.Context, bool) error { return d.File.Close() }

// OpenInstanceFile opens a package instance file on disk.
//
// The caller of this function must call closer() if err != nil to close the
// underlying file.
func OpenInstanceFile(ctx context.Context, path string, instanceID string, v VerificationMode) (inst PackageInstance, closer func() error, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	inst, err = OpenInstance(ctx, dummyInstance{file}, instanceID, v)
	if err != nil {
		inst = nil
		file.Close()
	} else {
		closer = file.Close
	}
	return
}

// ExtractFilter is predicate used by ExtractInstance to exclude files from
// extraction and the manifest.json. The function will be given the name of the
// file inside the instance zip (so, the path relative to the extraction
// destination).
type ExtractFilter func(f File) bool

// ExtractInstance extracts all files from a package instance into a destination.
//
// If exclude != nil, it will be used to filter the contents of the
// PackageInstance before extraction.
func ExtractInstance(ctx context.Context, inst PackageInstance, dest Destination, exclude ExtractFilter) error {
	if err := dest.Begin(ctx); err != nil {
		return err
	}

	// Do not leave garbage around in case of a panic.
	needToEnd := true
	defer func() {
		if needToEnd {
			dest.End(ctx, false)
		}
	}()

	allFiles := inst.Files()
	var files []File
	if exclude != nil {
		files = make([]File, 0, len(allFiles))
		for _, f := range allFiles {
			if !exclude(f) {
				files = append(files, f)
			}
		}
	} else {
		files = allFiles
	}

	progress := newProgressReporter(ctx, files)

	extractManifestFile := func(f File) (err error) {
		defer progress.advance(f)
		manifest, err := readManifestFile(f)
		if err != nil {
			return err
		}
		manifest.Files = make([]FileInfo, 0, len(files))
		for _, file := range files {
			// Do not put info about service .cipdpkg files into the manifest,
			// otherwise it becomes recursive and "size" property of manifest file
			// itself is not correct.
			if strings.HasPrefix(file.Name(), packageServiceDir+"/") {
				continue
			}
			fi := FileInfo{
				Name:       file.Name(),
				Size:       file.Size(),
				Executable: file.Executable(),
				WinAttrs:   file.WinAttrs().String(),
			}
			if file.Symlink() {
				target, err := file.SymlinkTarget()
				if err != nil {
					return err
				}
				fi.Symlink = target
			}
			manifest.Files = append(manifest.Files, fi)
		}
		out, err := dest.CreateFile(ctx, f.Name(), false, 0)
		if err != nil {
			return err
		}
		defer func() {
			if closeErr := out.Close(); err == nil {
				err = closeErr
			}
		}()
		return writeManifest(&manifest, out)
	}

	extractSymlinkFile := func(f File) error {
		defer progress.advance(f)
		target, err := f.SymlinkTarget()
		if err != nil {
			return err
		}
		return dest.CreateSymlink(ctx, f.Name(), target)
	}

	extractRegularFile := func(f File) (err error) {
		defer progress.advance(f)
		out, err := dest.CreateFile(ctx, f.Name(), f.Executable(), f.WinAttrs())
		if err != nil {
			return err
		}
		defer func() {
			if closeErr := out.Close(); err == nil {
				err = closeErr
			}
		}()
		in, err := f.Open()
		if err != nil {
			return err
		}
		defer in.Close()
		_, err = io.Copy(out, in)
		return err
	}

	var manifest File
	var err error
	for _, f := range files {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			break

		default:
			switch {
			case f.Name() == manifestName:
				// We delay writing the extended manifest until the very end because it
				// contains values of 'SymlinkTarget' fields of all extracted files.
				// Fetching 'SymlinkTarget' in general involves seeking inside the zip,
				// and we prefer not to do that now. Upon exit from the loop, all
				// 'SymlinkTarget' values will be already cached in memory, and writing
				// the manifest will be cheaper.
				manifest = f
			case f.Symlink():
				err = extractSymlinkFile(f)
			default:
				err = extractRegularFile(f)
			}
		}
		if err != nil {
			break
		}
	}

	// Finally extract the extended manifest, now that we have read (and cached)
	// all 'SymlinkTarget' values.
	if err == nil {
		if manifest == nil {
			err = fmt.Errorf("no %s file, this is bad", manifestName)
		} else {
			err = extractManifestFile(manifest)
		}
	}

	needToEnd = false
	if err == nil {
		err = dest.End(ctx, true)
	} else {
		// Ignore error in 'End' and return the original error.
		dest.End(ctx, false)
	}

	return err
}

// progressReporter periodically logs progress of the extraction.
//
// Can be shared by multiple goroutines.
type progressReporter struct {
	sync.Mutex

	ctx context.Context

	totalCount     uint64    // total number of files to extract
	totalSize      uint64    // total expected uncompressed size of files
	extractedCount uint64    // number of files extract so far
	extractedSize  uint64    // bytes uncompressed so far
	prevReport     time.Time // time when we did the last progress report
}

func newProgressReporter(ctx context.Context, files []File) *progressReporter {
	r := &progressReporter{ctx: ctx, totalCount: uint64(len(files))}
	for _, f := range files {
		if !f.Symlink() {
			r.totalSize += f.Size()
		}
	}
	if r.totalCount != 0 {
		logging.Infof(
			r.ctx, "cipd: about to extract %.1f Mb (%d files)",
			float64(r.totalSize)/1024.0/1024.0, r.totalCount)
	}
	return r
}

// advance moves the progress indicator, occasionally logging it.
func (r *progressReporter) advance(f File) {
	if r.totalCount == 0 {
		return
	}

	now := clock.Now(r.ctx)
	reportNow := false
	progress := 0

	// We don't count size of the symlinks toward total.
	var size uint64
	if !f.Symlink() {
		size = f.Size()
	}

	// Report progress on first and last 'advance' calls and each 2 sec.
	r.Lock()
	r.extractedSize += size
	r.extractedCount++
	if r.extractedCount == 1 || r.extractedCount == r.totalCount || now.Sub(r.prevReport) > 2*time.Second {
		reportNow = true
		if r.totalSize != 0 {
			progress = int(float64(r.extractedSize) * 100 / float64(r.totalSize))
		} else {
			progress = int(float64(r.extractedCount) * 100 / float64(r.totalCount))
		}
		r.prevReport = now
	}
	r.Unlock()

	if reportNow {
		logging.Infof(r.ctx, "cipd: extracting - %d%%", progress)
	}
}

////////////////////////////////////////////////////////////////////////////////
// PackageInstance implementation.

type packageInstance struct {
	data       InstanceFile
	instanceID string
	zip        *zip.Reader
	files      []File
	manifest   Manifest
}

// open reads the package data, verifies SHA1 hash and reads manifest.
//
// It doesn't check for corruption, but the caller must do so.
func (inst *packageInstance) open(instanceID string, v VerificationMode) error {
	var dataSize int64
	var err error

	switch {
	case instanceID == "":
		// Calculate the default hash and use it as instance ID, regardless of
		// the verification mode.
		h := common.DefaultHash()
		dataSize, err = getHashAndSize(inst.data, h)
		if err != nil {
			return err
		}
		instanceID = common.InstanceIDFromHash(h)

	case v == VerifyHash:
		var h hash.Hash
		h, err = common.HashForInstanceID(instanceID)
		if err != nil {
			return err
		}
		dataSize, err = getHashAndSize(inst.data, h)
		if err != nil {
			return err
		}
		if common.InstanceIDFromHash(h) != instanceID {
			return ErrHashMismatch
		}

	case v == SkipHashVerification:
		dataSize, err = inst.data.Seek(0, os.SEEK_END)
		if err != nil {
			return err
		}

	default:
		return fmt.Errorf("invalid verification mode %q", v)
	}

	// Assert it is well-formated. This is important for SkipHashVerification
	// mode, where the user can pass whatever.
	if err = common.ValidateInstanceID(instanceID); err != nil {
		return err
	}
	inst.instanceID = instanceID

	// Zip reader needs an io.ReaderAt. Try to sniff it from our io.ReadSeeker
	// before falling back to a generic (potentially slower) implementation. This
	// works if inst.data is actually an os.File (which happens quite often).
	reader, ok := inst.data.(io.ReaderAt)
	if !ok {
		reader = &readerAt{r: inst.data}
	}

	// List files and package manifest.
	inst.zip, err = zip.NewReader(reader, dataSize)
	if err != nil {
		return err
	}
	inst.files = make([]File, len(inst.zip.File))
	for i, zf := range inst.zip.File {
		fiz := &fileInZip{z: zf}
		if fiz.Name() == manifestName {
			// The manifest is later read again when extracting, keep it in memory.
			if err = fiz.prefetch(); err != nil {
				return err
			}
			if inst.manifest, err = readManifestFile(fiz); err != nil {
				return err
			}
		}
		inst.files[i] = fiz
	}

	// Generate version_file if needed.
	if inst.manifest.VersionFile != "" {
		vf, err := makeVersionFile(inst.manifest.VersionFile, VersionFile{
			PackageName: inst.manifest.PackageName,
			InstanceID:  inst.instanceID,
		})
		if err != nil {
			return err
		}
		inst.files = append(inst.files, vf)
	}

	return nil
}

func (inst *packageInstance) Pin() common.Pin {
	return common.Pin{
		PackageName: inst.manifest.PackageName,
		InstanceID:  inst.instanceID,
	}
}

func (inst *packageInstance) Files() []File             { return inst.files }
func (inst *packageInstance) DataReader() io.ReadSeeker { return inst.data }

// IsCorruptionError returns true iff err indicates corruption.
func IsCorruptionError(err error) bool {
	switch err {
	case zip.ErrFormat, zip.ErrChecksum, zip.ErrAlgorithm, ErrHashMismatch:
		return true
	}
	return false
}

////////////////////////////////////////////////////////////////////////////////
// Utilities.

// getHashAndSize rereads the entire file, passing it through the digester.
//
// Returns file length.
func getHashAndSize(r io.ReadSeeker, h hash.Hash) (int64, error) {
	if _, err := r.Seek(0, os.SEEK_SET); err != nil {
		return 0, err
	}
	if _, err := io.Copy(h, r); err != nil {
		return 0, err
	}
	return r.Seek(0, os.SEEK_CUR)
}

// readManifestFile decodes manifest file zipped inside the package.
func readManifestFile(f File) (Manifest, error) {
	r, err := f.Open()
	if err != nil {
		return Manifest{}, err
	}
	defer r.Close()
	return readManifest(r)
}

// makeVersionFile returns File representing a JSON blob with info about package
// version. It's what's deployed at path specified in 'version_file' stanza in
// package definition YAML.
func makeVersionFile(relPath string, versionFile VersionFile) (File, error) {
	if !isCleanSlashPath(relPath) {
		return nil, fmt.Errorf("invalid version_file: %s", relPath)
	}
	blob, err := json.MarshalIndent(versionFile, "", "  ")
	if err != nil {
		return nil, err
	}
	return &blobFile{
		name: relPath,
		blob: blob,
	}, nil
}

// blobFile implements File on top of byte array with file data.
type blobFile struct {
	name string
	blob []byte
}

func (b *blobFile) Name() string                   { return b.name }
func (b *blobFile) Size() uint64                   { return uint64(len(b.blob)) }
func (b *blobFile) Executable() bool               { return false }
func (b *blobFile) Symlink() bool                  { return false }
func (b *blobFile) SymlinkTarget() (string, error) { return "", nil }
func (b *blobFile) WinAttrs() WinAttrs             { return 0 }

func (b *blobFile) Open() (io.ReadCloser, error) {
	return ioutil.NopCloser(bytes.NewReader(b.blob)), nil
}

////////////////////////////////////////////////////////////////////////////////
// File interface implementation via zip.File.

type fileInZip struct {
	z    *zip.File
	body []byte // if not nil, uncompressed body of the file
}

// prefetch loads the body of file into memory to speed up later calls.
func (f *fileInZip) prefetch() error {
	if f.body != nil {
		return nil
	}
	r, err := f.z.Open()
	if err != nil {
		return err
	}
	defer r.Close()
	f.body, err = ioutil.ReadAll(r)
	return err
}

func (f *fileInZip) Name() string  { return f.z.Name }
func (f *fileInZip) Symlink() bool { return (f.z.Mode() & os.ModeSymlink) != 0 }
func (f *fileInZip) WinAttrs() WinAttrs {
	return WinAttrs(f.z.ExternalAttrs) & WinAttrsAll
}

func (f *fileInZip) Executable() bool {
	if f.Symlink() {
		return false
	}
	return (f.z.Mode() & 0100) != 0
}

func (f *fileInZip) Size() uint64 {
	if f.Symlink() {
		return 0
	}
	return f.z.UncompressedSize64
}

func (f *fileInZip) SymlinkTarget() (string, error) {
	if !f.Symlink() {
		return "", fmt.Errorf("not a symlink: %s", f.Name())
	}
	// Symlink is small, read it once and keep in memory. This is important
	// because 'SymlinkTarget' method looks like metadata getter, callers
	// don't expect it to do any IO each time (e.g. seeking inside the zip file).
	if err := f.prefetch(); err != nil {
		return "", err
	}
	return string(f.body), nil
}

func (f *fileInZip) Open() (io.ReadCloser, error) {
	if f.Symlink() {
		return nil, fmt.Errorf("opening a symlink is not allowed: %s", f.Name())
	}
	if f.body != nil {
		return ioutil.NopCloser(bytes.NewReader(f.body)), nil
	}
	return f.z.Open()
}

////////////////////////////////////////////////////////////////////////////////
// ReaderAt implementation via ReadSeeker. Not concurrency safe, moves file
// pointer around without any locking. Works OK in the context of OpenInstance
// function though (where OpenInstance takes sole ownership of io.ReadSeeker).

type readerAt struct {
	r io.ReadSeeker
}

func (r *readerAt) ReadAt(data []byte, off int64) (int, error) {
	_, err := r.r.Seek(off, os.SEEK_SET)
	if err != nil {
		return 0, err
	}
	return r.r.Read(data)
}
