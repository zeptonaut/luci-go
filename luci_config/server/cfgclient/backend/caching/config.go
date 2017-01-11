// Copyright 2015 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

package caching

import (
	"fmt"
	"net/url"

	"github.com/luci/luci-go/common/errors"
	log "github.com/luci/luci-go/common/logging"
	"github.com/luci/luci-go/luci_config/server/cfgclient"
	"github.com/luci/luci-go/luci_config/server/cfgclient/backend"

	"golang.org/x/net/context"
)

// Schema is the current package's cache schema.
const Schema = "v2"

// Operation is a cache entry operation. Cache entries are all stored in the
// same object, with different parameters filled based on the operation that
// they represent.
type Operation string

const (
	// OpGet is the Get operation.
	OpGet = Operation("Get")
	// OpGetAll is the GetAll operation.
	OpGetAll = Operation("GetAll")
	// OpConfigSetURL is the ConfigSetURL operation.
	OpConfigSetURL = Operation("ConfigSetURL")
)

// Key is a cache key.
type Key struct {
	// Schema is the schema for this key/value. If schema changes in a backwards-
	// incompatible way, this must also change.
	Schema string `json:"s,omitempty"`

	// Authority is the config authority to use.
	Authority backend.Authority `json:"a,omitempty"`

	// Op is the operation that is being cached.
	Op Operation `json:"op,omitempty"`

	// Content is true if this request asked for content.
	Content bool `json:"c,omitempty"`

	// Formatter is the requesting Key's Formatter parameter, expanded from its
	// FormatSpec.
	Formatter string `json:"f,omitempty"`
	// FormatData is the requesting Key's format Data parameter, expanded from its
	// FormatSpec.
	FormatData string `json:"fd,omitempty"`

	// ConfigSet is the config set parameter. This is valid for "OpGet".
	ConfigSet string `json:"cs,omitempty"`
	// Path is the path parameters. This is valid for "OpGet" and "OpGetAll".
	Path string `json:"p,omitempty"`

	// GetAllTarget is the "GetAll" operation type. This is valid for "OpGetAll".
	GetAllTarget backend.GetAllTarget `json:"gat,omitempty"`
}

// ParamHash returns a deterministic hash of all of the key parameters.
func (k *Key) ParamHash() []byte {
	cstr := ""
	if k.Content {
		cstr = "y"
	}
	return HashParams(k.Schema, string(k.Authority), string(k.Op), cstr, k.Formatter, k.FormatData,
		k.ConfigSet, k.Path, string(k.GetAllTarget))
}

// String prints a text representation of the key. No effort is made to ensure
// that this representation is consistent or deterministic, and it is not
// bound to the cache schema.
func (k *Key) String() string { return fmt.Sprintf("%#v", k) }

// Params returns the backend.Params that are encoded in this Key.
func (k *Key) Params() backend.Params {
	return backend.Params{
		Content:    k.Content,
		Authority:  k.Authority,
		FormatSpec: backend.FormatSpec{k.Formatter, k.FormatData},
	}
}

// ValueItem is a cache-optimized backend.Item projection.
//
// This will mostly be the same as a backend.Item, with the exception of
// Content, which will hold the formatted value if it has been formatted.
//
// See Formatter in
// github.com/luci/luci-go/luci_config/server/cfgclient
// and Backend in
// github.com/luci/luci-go/luci_config/server/cfgclient.backend/format
// for more information.
type ValueItem struct {
	ConfigSet string `json:"cs,omitempty"`
	Path      string `json:"p,omitempty"`

	ContentHash string `json:"ch,omitempty"`
	Revision    string `json:"r,omitempty"`

	Content []byte `json:"c,omitempty"`

	Formatter  string `json:"f,omitempty"`
	FormatData []byte `json:"fd,omitempty"`
}

// MakeValueItem builds a caching ValueItem from a backend.Item.
func MakeValueItem(it *backend.Item) ValueItem {
	return ValueItem{
		ConfigSet:   it.ConfigSet,
		Path:        it.Path,
		ContentHash: it.ContentHash,
		Revision:    it.Revision,
		Content:     []byte(it.Content),
		Formatter:   it.FormatSpec.Formatter,
		FormatData:  []byte(it.FormatSpec.Data),
	}
}

// ConfigItem returns the backend.Item equivalent of vi.
func (vi *ValueItem) ConfigItem() *backend.Item {
	return &backend.Item{
		Meta: backend.Meta{
			ConfigSet:   vi.ConfigSet,
			Path:        vi.Path,
			ContentHash: vi.ContentHash,
			Revision:    vi.Revision,
		},
		Content:    string(vi.Content),
		FormatSpec: backend.FormatSpec{vi.Formatter, string(vi.FormatData)},
	}
}

// Value is a cache value.
type Value struct {
	// Items is the cached set of config response items.
	//
	// For Get, this will either be empty (cached miss) or have a single Item
	// in it (cache hit).
	Items []ValueItem `json:"i,omitempty"`

	// URL is a URL string.
	//
	// Used with GetConfigSetURL.
	URL string `json:"u,omitempty"`
}

// LoadItems loads a set of backend.Item into v's Items field. If items is nil,
// v.Items will be nil.
func (v *Value) LoadItems(items ...*backend.Item) {
	if len(items) == 0 {
		v.Items = nil
		return
	}

	v.Items = make([]ValueItem, len(items))
	for i, it := range items {
		v.Items[i] = MakeValueItem(it)
	}
}

// SingleItem returns the first backend.Item in v's Items slice. If the Items
// slice is empty, SingleItem will return nil.
func (v *Value) SingleItem() *backend.Item {
	if len(v.Items) == 0 {
		return nil
	}
	return v.Items[0].ConfigItem()
}

// ConfigItems returns the backend.Item projection of v's Items slice.
func (v *Value) ConfigItems() []*backend.Item {
	if len(v.Items) == 0 {
		return nil
	}

	res := make([]*backend.Item, len(v.Items))
	for i := range v.Items {
		res[i] = v.Items[i].ConfigItem()
	}
	return res
}

// DecodeValue loads a Value from is encoded representation.
func DecodeValue(d []byte) (*Value, error) {
	var v Value
	if err := Decode(d, &v); err != nil {
		return nil, errors.Annotate(err).Err()
	}
	return &v, nil
}

// Encode encodes this Value.
//
// This is offered for convenience, but caches aren't required to use this
// encoding.
//
// The format stores the Value as compressed JSON.
func (v *Value) Encode() ([]byte, error) { return Encode(v) }

// Loader retrieves a Value by consulting the backing backend.
//
// The input Value is the current cached Value, or nil if there is no current
// cached Value. The output Value is the cached Value, if one exists. It is
// acceptable to return mutate "v" and/or return it as the output Value.
type Loader func(context.Context, Key, *Value) (*Value, error)

// Backend is a backend.B implementation that caches responses.
//
// All cached values are full-content regardless of whether or not full content
// was requested.
//
// Backend caches content and no-content requests as separate cache entries.
// This enables one cache to do low-overhead updating against another cache
// implementation.
type Backend struct {
	// Backend is the backing Backend.
	backend.B

	// FailOnError, if true, means that a failure to retrieve a cached item will
	// be propagated to the caller immediately. If false, a cache error will
	// result in a direct fall-through call to the embedded backend.B.
	//
	// One might set FailOnError to true if the fallback is unacceptably slow, or
	// if they want to enforce caching via failure.
	FailOnError bool

	// CacheGet retrieves the cached value associated with key. If no such cached
	// value exists, CacheGet is responsible for resolving the cache value using
	// the supplied Loader.
	CacheGet func(context.Context, Key, Loader) (*Value, error)
}

// Get implements backend.B.
func (b *Backend) Get(c context.Context, configSet, path string, p backend.Params) (*backend.Item, error) {
	key := Key{
		Schema:     Schema,
		Authority:  p.Authority,
		Op:         OpGet,
		Content:    p.Content,
		ConfigSet:  configSet,
		Path:       path,
		Formatter:  p.FormatSpec.Formatter,
		FormatData: p.FormatSpec.Data,
	}
	value, err := b.CacheGet(c, key, b.loader)
	if err != nil {
		if b.FailOnError {
			log.Fields{
				log.ErrorKey: err,
				"authority":  p.Authority,
				"configSet":  configSet,
				"path":       path,
			}.Errorf(c, "(Hard Failure) failed to load cache value.")
			return nil, errors.Annotate(err).Err()
		}

		log.Fields{
			log.ErrorKey: err,
			"authority":  p.Authority,
			"configSet":  configSet,
			"path":       path,
		}.Warningf(c, "Failed to load cache value.")
		return b.B.Get(c, configSet, path, p)
	}

	it := value.SingleItem()
	if it == nil {
		// Sentinel for no config.
		return nil, cfgclient.ErrNoConfig
	}
	return it, nil
}

// GetAll implements config.Backend.
func (b *Backend) GetAll(c context.Context, t backend.GetAllTarget, path string, p backend.Params) ([]*backend.Item, error) {
	key := Key{
		Schema:       Schema,
		Authority:    p.Authority,
		Op:           OpGetAll,
		Content:      p.Content,
		Path:         path,
		GetAllTarget: t,
		Formatter:    p.FormatSpec.Formatter,
		FormatData:   p.FormatSpec.Data,
	}
	value, err := b.CacheGet(c, key, b.loader)
	if err != nil {
		if b.FailOnError {
			log.Fields{
				log.ErrorKey: err,
				"authority":  p.Authority,
				"type":       t,
				"path":       path,
			}.Errorf(c, "(Hard Failure) failed to load cache value.")
			return nil, errors.Annotate(err).Err()
		}

		log.Fields{
			log.ErrorKey: err,
			"authority":  p.Authority,
			"type":       t,
			"path":       path,
		}.Warningf(c, "Failed to load cache value.")
		return b.B.GetAll(c, t, path, p)
	}
	return value.ConfigItems(), nil
}

// ConfigSetURL implements backend.B.
func (b *Backend) ConfigSetURL(c context.Context, configSet string, p backend.Params) (u url.URL, err error) {
	key := Key{
		Schema:    Schema,
		Authority: p.Authority,
		Op:        OpConfigSetURL,
		ConfigSet: configSet,
	}

	var value *Value
	if value, err = b.CacheGet(c, key, b.loader); err != nil {
		if b.FailOnError {
			log.Fields{
				log.ErrorKey: err,
				"authority":  p.Authority,
				"configSet":  configSet,
			}.Errorf(c, "(Hard Failure) failed to load cache value.")
			err = errors.Annotate(err).Err()
			return
		}

		log.Fields{
			log.ErrorKey: err,
			"authority":  p.Authority,
			"configSet":  configSet,
		}.Warningf(c, "Failed to load cache value.")
		return b.B.ConfigSetURL(c, configSet, p)
	}

	if value.URL == "" {
		// Sentinel for no config.
		err = cfgclient.ErrNoConfig
		return
	}

	up, err := url.Parse(value.URL)
	if err != nil {
		err = errors.Annotate(err).Reason("failed to parse cached URL: %(value)q").D("value", value.URL).Err()
		return
	}

	u = *up
	return
}

// loader runs a cache get against the configured Base backend.
//
// This should be used by caches that do not have the cached value.
func (b *Backend) loader(c context.Context, k Key, v *Value) (*Value, error) {
	return CacheLoad(c, b.B, k, v)
}

// CacheLoad loads k from backend b.
//
// If an existing cache value is known, it should be supplied as v. Otherwise,
// v should be nil.
//
// This is effectively a Loader function that is detached from a given cache
// instance.
func CacheLoad(c context.Context, b backend.B, k Key, v *Value) (rv *Value, err error) {
	switch k.Op {
	case OpGet:
		rv, err = doGet(c, b, k.ConfigSet, k.Path, v, k.Params())
	case OpGetAll:
		rv, err = doGetAll(c, b, k.GetAllTarget, k.Path, v, k.Params())
	case OpConfigSetURL:
		rv, err = doConfigSetURL(c, b, k.ConfigSet, k.Params())
	default:
		return nil, errors.Reason("unknown operation: %(op)v").D("op", k.Op).Err()
	}
	if err != nil {
		return nil, err
	}
	return
}

func doGet(c context.Context, b backend.B, configSet, path string, v *Value, p backend.Params) (*Value, error) {
	hadItem := (v != nil && len(v.Items) > 0)
	if !hadItem {
		// Initialize empty "v".
		v = &Value{}
	}

	// If we have a current item, or if we are requesting "no-content", then
	// perform a "no-content" lookup.
	if hadItem || !p.Content {
		noContentP := p
		noContentP.Content = false

		item, err := b.Get(c, configSet, path, noContentP)
		switch err {
		case nil:
			if hadItem && (item.ContentHash == v.Items[0].ContentHash) {
				// Nothing changed.
				return v, nil
			}

			// If our "Get" is, itself, no-content, then this is our actual result.
			if !p.Content {
				v.LoadItems(item)
				return v, nil
			}

			// If we get here, we are requesting full-content and our hash check
			// showed that the current content has a different hash.
			break

		case cfgclient.ErrNoConfig:
			v.LoadItems()
			return v, nil

		default:
			return nil, errors.Annotate(err).Err()
		}
	}

	// Perform a full content request.
	switch item, err := b.Get(c, configSet, path, p); err {
	case nil:
		v.LoadItems(item)
		return v, nil

	case cfgclient.ErrNoConfig:
		// Empty "config missing" item.
		v.LoadItems()
		return v, nil

	default:
		return nil, errors.Annotate(err).Err()
	}
}

func doGetAll(c context.Context, b backend.B, t backend.GetAllTarget, path string, v *Value, p backend.Params) (
	*Value, error) {

	// If we already have a cached value, or if we're requesting no-content, do a
	// no-content refresh to see if anything has changed.
	//
	// Response values are in order, so this is a simple traversal.
	if v != nil || !p.Content {
		noContentP := p
		noContentP.Content = false

		items, err := b.GetAll(c, t, path, noContentP)
		if err != nil {
			return nil, errors.Annotate(err).Reason("failed RPC (hash-only)").Err()
		}

		// If we already have a cached item, validate it.
		if v != nil && len(items) == len(v.Items) {
			match := true
			for i, other := range items {
				cur := v.Items[i]
				if cur.ConfigSet == other.ConfigSet && cur.Path == other.Path &&
					cur.ContentHash == other.ContentHash {
					continue
				}

				match = false
				break
			}

			// If all configs match, our response hasn't changed.
			if match {
				return v, nil
			}
		}

		// If we requested no-content, then this is our result.
		if !p.Content {
			var retV Value
			retV.LoadItems(items...)
			return &retV, nil
		}
	}

	// Perform a full-content request.
	items, err := b.GetAll(c, t, path, p)
	if err != nil {
		return nil, errors.Annotate(err).Err()
	}
	var retV Value
	retV.LoadItems(items...)
	return &retV, nil
}

func doConfigSetURL(c context.Context, b backend.B, configSet string, p backend.Params) (*Value, error) {
	u, err := b.ConfigSetURL(c, configSet, p)
	switch err {
	case nil:
		return &Value{
			URL: u.String(),
		}, nil

	case cfgclient.ErrNoConfig:
		return &Value{}, nil

	default:
		return nil, err
	}
}