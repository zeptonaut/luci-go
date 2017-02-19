// Copyright 2015 The LUCI Authors. All rights reserved.
// Use of this source code is governed under the Apache License, Version 2.0
// that can be found in the LICENSE file.

// Package auth implements a wrapper around golang.org/x/oauth2.
//
// Its main improvement is the on-disk cache for OAuth tokens, which is
// especially important for 3-legged interactive OAuth flows: its usage
// eliminates annoying login prompts each time a program is used (because the
// refresh token can now be reused). The cache also allows to reduce unnecessary
// token refresh calls when sharing a service account between processes.
//
// The package also implements some best practices regarding interactive login
// flows in CLI programs. It makes it easy to implement a login process as
// a separate interactive step that happens before the main program loop.
//
// The antipattern it tries to prevent is "launch an interactive login flow
// whenever program hits 'Not Authorized' response from the server". This
// usually results in a very confusing behavior, when login prompts pop up
// unexpectedly at random time, random places and from multiple goroutines at
// once, unexpectedly consuming unintended stdin input.
package auth

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/compute/metadata"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/credentials"

	"github.com/luci/luci-go/common/auth/internal"
	"github.com/luci/luci-go/common/clock"
	"github.com/luci/luci-go/common/errors"
	"github.com/luci/luci-go/common/gcloud/iam"
	"github.com/luci/luci-go/common/logging"
	"github.com/luci/luci-go/common/retry"
	"github.com/luci/luci-go/lucictx"
)

var (
	// ErrLoginRequired is returned by Transport() in case long term credentials
	// are not cached and the user must go through interactive login.
	ErrLoginRequired = errors.New("interactive login is required")

	// ErrInsufficientAccess is returned by Login() or Transport() if access_token
	// can't be minted for given OAuth scopes. For example if GCE instance wasn't
	// granted access to requested scopes when it was created.
	ErrInsufficientAccess = internal.ErrInsufficientAccess

	// ErrBadCredentials is returned by authenticating RoundTripper if service
	// account key used to generate access tokens is revoked, malformed or can not
	// be read from disk.
	ErrBadCredentials = internal.ErrBadCredentials
)

// Known Google API OAuth scopes.
const (
	OAuthScopeEmail = "https://www.googleapis.com/auth/userinfo.email"
)

// Method defines a method to use to obtain OAuth access token.
type Method string

// Supported authentication methods.
const (
	// AutoSelectMethod can be used to allow the library to pick a method most
	// appropriate for given set of options and the current execution environment.
	//
	// For example, passing ServiceAccountJSONPath or ServiceAcountJSON makes
	// Authenticator to pick ServiceAccountMethod.
	//
	// See SelectBestMethod function for details.
	AutoSelectMethod Method = ""

	// UserCredentialsMethod is used for interactive OAuth 3-legged login flow.
	//
	// Using this method requires specifying an OAuth client by passing ClientID
	// and ClientSecret in Options when calling NewAuthenticator.
	//
	// Additionally, SilentLogin and OptionalLogin (i.e. non-interactive) login
	// modes rely on a presence of a refresh token in the token cache, thus using
	// these modes with UserCredentialsMethod also requires configured token
	// cache (see SecretsDir field of Options).
	UserCredentialsMethod Method = "UserCredentialsMethod"

	// ServiceAccountMethod is used to authenticate as a service account using
	// a private key.
	//
	// Callers of NewAuthenticator must pass either a path to a JSON file with
	// service account key (as produced by Google Cloud Console) or a body of this
	// JSON file. See ServiceAccountJSONPath and ServiceAccountJSON fields in
	// Options.
	//
	// Using ServiceAccountJSONPath has an advantage: Authenticator always loads
	// the private key from the file before refreshing the token, it allows to
	// replace the key while the process is running.
	ServiceAccountMethod Method = "ServiceAccountMethod"

	// GCEMetadataMethod is used on Compute Engine to use tokens provided by
	// Metadata server. See https://cloud.google.com/compute/docs/authentication
	GCEMetadataMethod Method = "GCEMetadataMethod"

	// LUCIContextMethod is used by LUCI-aware applications to fetch tokens though
	// a local auth server (discoverable via "local_auth" key in LUCI_CONTEXT).
	//
	// This method is similar in spirit to GCEMetadataMethod: it uses some local
	// HTTP server as a provider of OAuth access tokens, which gives an ambient
	// authentication context to apps that use it.
	//
	// There are some big differences:
	//  1. LUCIContextMethod supports minting tokens for multiple different set
	//     of scopes, unlike GCE metadata server that always gives a token with
	//     preconfigured scopes (set when the GCE instance was created).
	//  2. LUCIContextMethod is not GCE-specific. It doesn't use magic link-local
	//     IP address. It can run on any machine.
	//  3. The access to the local auth server is controlled by file system
	//     permissions of LUCI_CONTEXT file (there's a secret in this file).
	//  4. There can be many local auth servers running at once (on different
	//     ports). Useful for bringing up sub-contexts, in particular in
	//     combination with ActAsServiceAcccount ("sudo" mode) or for tests.
	//
	// See common/auth/localauth package for the implementation of the server side
	// of the protocol.
	LUCIContextMethod Method = "LUCIContextMethod"
)

// LoginMode is used as enum in NewAuthenticator function.
type LoginMode string

const (
	// InteractiveLogin instructs Authenticator to ignore cached tokens (if any)
	// and forcefully rerun interactive login flow in Transport(), Client() and
	// other factories or Login() (whichever is called first).
	//
	// This is typically used with UserCredentialsMethod to generate an OAuth
	// refresh token and put it in the token cache, to make it available for later
	// non-interactive usage.
	//
	// When used with service account credentials (ServiceAccountMethod mode),
	// just precaches the access token.
	InteractiveLogin LoginMode = "InteractiveLogin"

	// SilentLogin indicates to Authenticator that it must return a transport that
	// implements authentication, but it is NOT OK to run interactive login flow
	// to make it.
	//
	// Transport() and other factories will fail with ErrLoginRequired error if
	// there's no cached token or one can't be generated on the fly in
	// non-interactive mode. This may happen when using UserCredentialsMethod.
	//
	// It is always OK to use SilentLogin mode with service accounts credentials
	// (ServiceAccountMethod mode), since no user interaction is necessary to
	// generate an access token in this case.
	SilentLogin LoginMode = "SilentLogin"

	// OptionalLogin indicates to Authenticator that it should return a transport
	// that implements authentication, but it is OK to return non-authenticating
	// transport if there are no valid cached credentials.
	//
	// An interactive login flow will never be invoked. An unauthenticated client
	// will be returned if no credentials are present.
	//
	// Can be used when making calls to backends that allow anonymous access. This
	// is especially useful with UserCredentialsMethod: a user may start using
	// the service right away (in anonymous mode), and later login (using Login()
	// method or any other way of initializing credentials cache) to get more
	// permissions.
	//
	// TODO(vadimsh): When used with ServiceAccountMethod it is identical to
	// SilentLogin, since it makes no sense to ignore invalid service account
	// credentials when the caller is explicitly asking the authenticator to use
	// them.
	//
	// Has the original meaning when used with GCEMetadataMethod: it instructs to
	// skip authentication if the token returned by GCE metadata service doesn't
	// have all requested scopes.
	OptionalLogin LoginMode = "OptionalLogin"
)

// minAcceptedLifetime is minimal lifetime of a token returned by the token
// source or put into authentication headers.
//
// If token is expected to live less than this duration, it will be refreshed.
const minAcceptedLifetime = 2 * time.Minute

// Options are used by NewAuthenticator call.
type Options struct {
	// Transport is underlying round tripper to use for requests.
	//
	// Default: http.DefaultTransport.
	Transport http.RoundTripper

	// Method defines how to grab OAuth2 tokens.
	//
	// Default: AutoSelectMethod.
	Method Method

	// Scopes is a list of OAuth scopes to request.
	//
	// Default: [OAuthScopeEmail].
	Scopes []string

	// ActAsServiceAccount is used to act as a specified service account email.
	//
	// This uses signBlob Cloud IAM API and "iam.serviceAccountActor" role.
	//
	// When this option is set, there are two identities involved:
	//  1. A service account identity directly specified by `ActAsServiceAccount`.
	//  2. An identity conveyed by the authenticator options (via cached refresh
	//     token, or via `ServiceAccountJSON`, or other similar ways), i.e. the
	//     identity asserted by the authenticator in case `ActAsServiceAccount` is
	//     not set. This identity must have "iam.serviceAccountActor" role in
	//     the `ActAsServiceAccount` IAM resource. It is referred to below as
	//     Actor identity.
	//
	// The resulting authenticator will produce access tokens for service account
	// `ActAsServiceAccount`, using Actor identity to generate them via Cloud IAM
	// API.
	//
	// `Scopes` parameter specifies what OAuth scopes to request for access tokens
	// belonging to `ActAsServiceAccount`.
	//
	// The Actor credentials will be internally used to generate access token with
	// IAM scope ("https://www.googleapis.com/auth/iam"). It means Login() action
	// sets up a refresh token with IAM scope (not `Scopes`), and the user will
	// be presented with a consent screen for IAM scope.
	//
	// More info at https://cloud.google.com/iam/docs/service-accounts
	//
	// Default: none.
	ActAsServiceAccount string

	// ClientID is OAuth client ID to use with UserCredentialsMethod.
	//
	// See https://developers.google.com/identity/protocols/OAuth2InstalledApp
	// (in particular everything related to "Desktop apps").
	//
	// Together with Scopes forms a cache key in the token cache, which in
	// practical terms means there can be only one concurrently "logged in" user
	// per [ClientID, Scopes] combination. So if multiple binaries use exact same
	// ClientID and Scopes, they'll share credentials cache (a login in one app
	// makes the user logged in in the other app too).
	//
	// If you don't want to share login information between tools, use separate
	// ClientID or SecretsDir values.
	//
	// If not set, UserCredentialsMethod auth method will not work.
	//
	// Default: none.
	ClientID string

	// ClientSecret is OAuth client secret to use with UserCredentialsMethod.
	//
	// Default: none.
	ClientSecret string

	// ServiceAccountJSONPath is a path to a JSON blob with a private key to use.
	//
	// Used only with ServiceAccountMethod.
	ServiceAccountJSONPath string

	// ServiceAccountJSON is a body of JSON key file to use.
	//
	// Overrides ServiceAccountJSONPath if given.
	ServiceAccountJSON []byte

	// GCEAccountName is an account name to query to fetch token for from metadata
	// server when GCEMetadataMethod is used.
	//
	// If given account wasn't granted required set of scopes during instance
	// creation time, Transport() call fails with ErrInsufficientAccess.
	//
	// Default: "default" account.
	GCEAccountName string

	// SecretsDir can be used to set the path to a directory where tokens
	// are cached.
	//
	// If not set, tokens will be cached only in the process memory. For refresh
	// tokens it means the user would have to go through the login process each
	// time process is started. For service account tokens it means there'll be
	// HTTP round trip to OAuth backend to generate access token each time the
	// process is started.
	SecretsDir string

	// DisableMonitoring can be used to disable the monitoring instrumentation.
	//
	// The transport produced by this authenticator sends tsmon metrics IFF:
	//  1. DisableMonitoring is false (default).
	//  2. The context passed to 'NewAuthenticator' has monitoring initialized.
	DisableMonitoring bool

	// MonitorAs is used for 'client' field of monitoring metrics.
	//
	// The default is 'luci-go'.
	MonitorAs string

	// testingBaseTokenProvider is used in unit tests.
	testingBaseTokenProvider internal.TokenProvider
	// testingIAMTokenProvider is used in unit tests.
	testingIAMTokenProvider internal.TokenProvider
}

// SelectBestMethod returns a most appropriate authentication method for the
// given set of options and the current execution environment.
//
// Invoked by Authenticator if AutoSelectMethod is passed as Method in Options.
// It picks the first applicable method in this order:
//   * ServiceAccountMethod (if the service account private key is configured).
//   * LUCIContextMethod (if running inside LUCI_CONTEXT with an auth server).
//   * GCEMetadataMethod (if running on GCE).
//   * UserCredentialsMethod (if no other method applies).
//
// Beware: it may do relatively heavy calls on first usage (to detect GCE
// environment). Fast after that.
func SelectBestMethod(ctx context.Context, opts Options) Method {
	switch {
	case opts.ServiceAccountJSONPath != "" || len(opts.ServiceAccountJSON) != 0:
		return ServiceAccountMethod
	case lucictx.GetLocalAuth(ctx) != nil:
		return LUCIContextMethod
	case metadata.OnGCE():
		return GCEMetadataMethod
	default:
		return UserCredentialsMethod
	}
}

// AllowsArbitraryScopes returns true if given authenticator options allow
// generating tokens for arbitrary set of scopes.
//
// For example, using a private key to sign assertions allows to mint tokens
// for any set of scopes (since there's no restriction on what scopes we can
// put into JWT to be signed).
//
// On other hand, using e.g GCE metadata server restricts us to use only scopes
// assigned to GCE instance when it was created.
func AllowsArbitraryScopes(ctx context.Context, opts Options) bool {
	if opts.Method == AutoSelectMethod {
		opts.Method = SelectBestMethod(ctx, opts)
	}
	switch {
	case opts.Method == ServiceAccountMethod:
		// A private key can be used to generate tokens with any combination of
		// scopes.
		return true
	case opts.Method == LUCIContextMethod:
		// We can ask the local auth server for any combination of scopes.
		return true
	case opts.ActAsServiceAccount != "":
		// When using IAM-derived tokens authenticator relies on singBytes IAM RPC.
		// It is similar to having a private key, and also can be used to generate
		// tokens with any combination of scopes
		return true
	}
	return false
}

// Authenticator is a factory for http.RoundTripper objects that know how to use
// cached OAuth credentials and how to send monitoring metrics (if tsmon package
// was imported).
//
// Authenticator also knows how to run interactive login flow, if required.
type Authenticator struct {
	// Immutable members.
	loginMode    LoginMode
	opts         *Options
	transport    http.RoundTripper
	ctx          context.Context
	testingCache internal.TokenCache // set in unit tests

	// Mutable members.
	lock sync.RWMutex
	err  error

	// baseToken is a token (and its provider and cache) whose possession is
	// sufficient to get the final access token used for authentication of user
	// calls (see 'authToken' below).
	//
	// Methods like 'CheckLoginRequired' check that the base token exists in the
	// cache or can be generated on the fly.
	//
	// In actor mode, the base token is always an IAM-scoped token that is used
	// to call signBlob API to generate an auth token. Base token is also always
	// using whatever auth method was specified by Options.Method.
	//
	// In non-actor mode, baseToken coincides with authToken: both point to exact
	// same struct.
	baseToken *tokenWithProvider

	// authToken is a token (and its provider and cache) that is actually used for
	// authentication of user calls.
	//
	// It is a token returned by 'GetAccessToken'. It is always scoped to 'Scopes'
	// list, as passed to NewAuthenticator via Options.
	//
	// In actor mode, it is derived from the base token by using SignBlob IAM API.
	// This process is non-interactive and thus can always be performed as long
	// as we have the base token.
	//
	// In non-actor mode it is the main token generated by the authenticator. In
	// this case it coincides with baseToken: both point to exact same object.
	authToken *tokenWithProvider
}

// NewAuthenticator returns a new instance of Authenticator given its options.
//
// The authenticator is essentially a factory for http.RoundTripper that knows
// how to use OAuth2 tokens. It is bound to the given context: uses its logger,
// clock, transport and deadline.
func NewAuthenticator(ctx context.Context, loginMode LoginMode, opts Options) *Authenticator {
	// Add default scope, sort scopes.
	if len(opts.Scopes) == 0 {
		opts.Scopes = []string{OAuthScopeEmail}
	} else {
		opts.Scopes = append([]string(nil), opts.Scopes...) // copy
		sort.Strings(opts.Scopes)
	}

	// Fill in blanks with default values.
	if opts.GCEAccountName == "" {
		opts.GCEAccountName = "default"
	}
	if opts.Transport == nil {
		opts.Transport = http.DefaultTransport
	}

	// TODO(vadimsh): Check SecretsDir permissions. It should be 0700.
	if opts.SecretsDir != "" && !filepath.IsAbs(opts.SecretsDir) {
		var err error
		opts.SecretsDir, err = filepath.Abs(opts.SecretsDir)
		if err != nil {
			panic(fmt.Errorf("failed to get abs path to token cache dir: %s", err))
		}
	}

	// See ensureInitialized for the rest of the initialization.
	auth := &Authenticator{
		ctx:       ctx,
		loginMode: loginMode,
		opts:      &opts,
	}
	auth.transport = NewModifyingTransport(opts.Transport, auth.authTokenInjector)

	// Include the token refresh time into the monitored request time.
	if globalInstrumentTransport != nil && !opts.DisableMonitoring {
		monitorAs := opts.MonitorAs
		if monitorAs == "" {
			monitorAs = "luci-go"
		}
		instrumented := globalInstrumentTransport(ctx, auth.transport, monitorAs)
		if instrumented != auth.transport {
			logging.Debugf(ctx, "Enabling monitoring instrumentation (client == %q)", monitorAs)
			auth.transport = instrumented
		}
	}

	return auth
}

// Transport optionally performs a login and returns http.RoundTripper.
//
// It is a high level wrapper around CheckLoginRequired() and Login() calls. See
// documentation for LoginMode for more details.
func (a *Authenticator) Transport() (http.RoundTripper, error) {
	switch useAuth, err := a.doLoginIfRequired(false); {
	case err != nil:
		return nil, err
	case useAuth:
		return a.transport, nil // token-injecting transport
	default:
		return a.opts.Transport, nil // original non-authenticating transport
	}
}

// Client optionally performs a login and returns http.Client.
//
// It uses transport returned by Transport(). See documentation for LoginMode
// for more details.
func (a *Authenticator) Client() (*http.Client, error) {
	transport, err := a.Transport()
	if err != nil {
		return nil, err
	}
	return &http.Client{Transport: transport}, nil
}

// TokenSource optionally performs a login and returns oauth2.TokenSource.
//
// Can be used for interoperability with libraries that use golang.org/x/oauth2.
//
// It doesn't support 'OptionalLogin' mode, since oauth2.TokenSource must return
// some token. Otherwise its logic is similar to Transport(). In particular it
// may return ErrLoginRequired if interactive login is required, but the
// authenticator is in the silent mode. See LoginMode enum for more details.
func (a *Authenticator) TokenSource() (oauth2.TokenSource, error) {
	if _, err := a.doLoginIfRequired(true); err != nil {
		return nil, err
	}
	return tokenSource{a}, nil
}

// PerRPCCredentials optionally performs a login and returns PerRPCCredentials.
//
// It can be used to authenticate outbound gPRC RPC's.
//
// Has same logic as Transport(), in particular supports OptionalLogin mode.
// See Transport() for more details.
func (a *Authenticator) PerRPCCredentials() (credentials.PerRPCCredentials, error) {
	switch useAuth, err := a.doLoginIfRequired(false); {
	case err != nil:
		return nil, err
	case useAuth:
		return perRPCCreds{a}, nil // token-injecting PerRPCCredentials
	default:
		return perRPCCreds{}, nil // noop PerRPCCredentials
	}
}

// GetAccessToken returns a valid access token with specified minimum lifetime.
//
//
// Does not interact with the user. May return ErrLoginRequired.
func (a *Authenticator) GetAccessToken(lifetime time.Duration) (*oauth2.Token, error) {
	tok, err := a.currentToken()
	if err != nil {
		return nil, err
	}
	if tok == nil || internal.TokenExpiresInRnd(a.ctx, tok, lifetime) {
		var err error
		tok, err = a.refreshToken(tok, lifetime)
		if err != nil {
			return nil, err
		}
		// Note: no randomization here. It is a sanity check that verifies
		// refreshToken did its job.
		if internal.TokenExpiresIn(a.ctx, tok, lifetime) {
			return nil, fmt.Errorf("auth: failed to refresh the token")
		}
	}
	return tok, nil
}

// CheckLoginRequired decides whether an interactive login is required.
//
// It examines the token cache and the configured authentication method to
// figure out whether we can attempt to grab an access token without involving
// the user interaction.
//
// Note: it does not check that the cached refresh token is still valid (i.e.
// not revoked). A revoked token will result in ErrLoginRequired error on a
// first attempt to use it.
//
// Returns:
//   * nil if we have a valid cached token or can mint one on the fly.
//   * ErrLoginRequired if we have no cached token and need to bother the user.
//   * ErrInsufficientAccess if the configured auth method can't mint the token
//     we require (e.g when using GCE method and the instance doesn't have all
//     requested OAuth scopes).
//   * Generic error on other unexpected errors.
func (a *Authenticator) CheckLoginRequired() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if err := a.ensureInitialized(); err != nil {
		return err
	}

	// No cached base token and the token provider requires interaction with the
	// user: need to login. Only non-interactive token providers are allowed to
	// mint tokens on the fly, see refreshToken.
	if a.baseToken.token == nil && a.baseToken.provider.RequiresInteraction() {
		return ErrLoginRequired
	}

	return nil
}

// Login perform an interaction with the user to get a long term refresh token
// and cache it.
//
// Blocks for user input, can use stdin. It overwrites currently cached
// credentials, if any.
func (a *Authenticator) Login() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	err := a.ensureInitialized()
	if err != nil {
		return err
	}
	if !a.baseToken.provider.RequiresInteraction() {
		return nil // can mint the token on the fly, no need for login
	}

	// Create an initial base token. This may require interaction with a user. Do
	// not do retries here, since Login is called when the user is looking, let
	// the user do the retries (since if MintToken() interacts with the user,
	// retrying it automatically will be extra confusing).
	a.baseToken.token, err = a.baseToken.provider.MintToken(a.ctx, nil)
	if err != nil {
		return err
	}

	// Store the initial token in the cache. Don't abort if it fails, the token
	// is still usable from the memory.
	if err := a.baseToken.putToCache(a.ctx); err != nil {
		logging.Warningf(a.ctx, "Failed to write token to cache: %s", err)
	}

	return nil
}

// PurgeCredentialsCache removes cached tokens.
//
// Does not revoke them!
func (a *Authenticator) PurgeCredentialsCache() error {
	a.lock.Lock()
	defer a.lock.Unlock()
	if err := a.ensureInitialized(); err != nil {
		return err
	}

	// No need to purge twice if baseToken == authToken, which is the case in
	// non-actor mode.
	var merr errors.MultiError
	if a.baseToken != a.authToken {
		merr = errors.NewMultiError(
			a.baseToken.purgeToken(a.ctx),
			a.authToken.purgeToken(a.ctx))
	} else {
		merr = errors.NewMultiError(a.baseToken.purgeToken(a.ctx))
	}

	switch total, first := merr.Summary(); {
	case total == 0:
		return nil
	case total == 1:
		return first
	default:
		return merr
	}
}

////////////////////////////////////////////////////////////////////////////////
// credentials.PerRPCCredentials implementation.

type perRPCCreds struct {
	a *Authenticator
}

func (creds perRPCCreds) GetRequestMetadata(c context.Context, uri ...string) (map[string]string, error) {
	if len(uri) == 0 {
		panic("perRPCCreds: no URI given")
	}
	if creds.a == nil {
		return nil, nil
	}
	tok, err := creds.a.GetAccessToken(minAcceptedLifetime)
	if err != nil {
		return nil, err
	}
	return map[string]string{
		"Authorization": tok.TokenType + " " + tok.AccessToken,
	}, nil
}

func (creds perRPCCreds) RequireTransportSecurity() bool { return true }

////////////////////////////////////////////////////////////////////////////////
// oauth2.TokenSource implementation.

type tokenSource struct {
	a *Authenticator
}

// Token is part of oauth2.TokenSource inteface.
func (s tokenSource) Token() (*oauth2.Token, error) {
	return s.a.GetAccessToken(minAcceptedLifetime)
}

////////////////////////////////////////////////////////////////////////////////
// Authenticator private methods.

// isActing is true if ActAsServiceAccount is set.
//
// In this mode baseToken != authToken.
func (a *Authenticator) isActing() bool {
	return a.opts.ActAsServiceAccount != ""
}

// checkInitialized is (true, <err>) if initialization happened (successfully or
// not) of (false, nil) if not.
func (a *Authenticator) checkInitialized() (bool, error) {
	if a.err != nil || a.baseToken != nil {
		return true, a.err
	}
	return false, nil
}

// ensureInitialized instantiates TokenProvider and reads token from cache.
//
// It is supposed to be called under the lock.
func (a *Authenticator) ensureInitialized() error {
	// Already initialized (successfully or not)?
	if initialized, err := a.checkInitialized(); initialized {
		return err
	}

	// SelectBestMethod may do heavy calls like talking to GCE metadata server,
	// call it lazily here rather than in NewAuthenticator.
	if a.opts.Method == AutoSelectMethod {
		a.opts.Method = SelectBestMethod(a.ctx, *a.opts)
	}

	// In Actor mode, make the base token IAM-scoped, to be able to use SignBlob
	// API. In non-actor mode, the base token is also the main auth token, so
	// scope it to whatever options were requested.
	scopes := a.opts.Scopes
	if a.isActing() {
		scopes = []string{iam.OAuthScope}
	}
	a.baseToken = &tokenWithProvider{}
	a.baseToken.provider, a.err = makeBaseTokenProvider(a.ctx, a.opts, scopes)
	if a.err != nil {
		return a.err // note: this can be ErrInsufficientAccess
	}

	// In non-actor mode, the token we must check in 'CheckLoginRequired' is the
	// same as returned by 'GetAccessToken'. In actor mode, they are different.
	// See comments for 'baseToken' and 'authToken'.
	if a.isActing() {
		a.authToken = &tokenWithProvider{}
		a.authToken.provider, a.err = makeIAMTokenProvider(a.ctx, a.opts)
		if a.err != nil {
			return a.err
		}
	} else {
		a.authToken = a.baseToken
	}

	// Initialize the token cache. Use the disk cache only if SecretsDir is given
	// and any of the providers is not "lightweight" (so it makes sense to
	// actually hit the disk, rather then call the provider each time new token is
	// needed).
	//
	// Note also that tests set a.testingCache before ensureInitialized() is
	// called to mock the cache. Respect this.
	if a.testingCache != nil {
		a.baseToken.cache = a.testingCache
		a.authToken.cache = a.testingCache
	} else {
		cache := internal.ProcTokenCache
		if !a.baseToken.provider.Lightweight() || !a.authToken.provider.Lightweight() {
			if a.opts.SecretsDir != "" {
				cache = &internal.DiskTokenCache{
					Context:    a.ctx,
					SecretsDir: a.opts.SecretsDir,
				}
			} else {
				logging.Warningf(a.ctx, "Disabling auth disk token cache. Not configured.")
			}
		}
		// Use the disk cache only for non-lightweight providers to avoid
		// unnecessarily leaks of tokens to the disk.
		if a.baseToken.provider.Lightweight() {
			a.baseToken.cache = internal.ProcTokenCache
		} else {
			a.baseToken.cache = cache
		}
		if a.authToken.provider.Lightweight() {
			a.authToken.cache = internal.ProcTokenCache
		} else {
			a.authToken.cache = cache
		}
	}

	// Interactive providers need to know whether there's a cached token (to ask
	// to run interactive login if there's none). Non-interactive providers do not
	// care about state of the cache that much (they know how to update it
	// themselves). So examine the cache here only when using interactive
	// provider. Non interactive providers will do it lazily on a first
	// refreshToken(...) call.
	if a.baseToken.provider.RequiresInteraction() {
		// Broken token cache is not a fatal error. So just log it and forget, a new
		// token will be minted in Login.
		if err := a.baseToken.fetchFromCache(a.ctx); err != nil {
			logging.Warningf(a.ctx, "Failed to read auth token from cache: %s", err)
		}
	}

	// Note: a.authToken.provider is either equal to a.baseToken.provider (if not
	// using actor mode), or (when using actor mode) it doesn't require
	// interaction (because it is an IAM one). So don't bother with fetching
	// 'authToken' from cache. It will be fetched lazily on the first use.

	return nil
}

// doLoginIfRequired optionally performs an interactive login.
//
// This is the main place where LoginMode handling is performed. Used by various
// factories (Transport, PerRPCCredentials, TokenSource, ...).
//
// If requiresAuth is false, we respect OptionalLogin mode. If true - we treat
// OptionalLogin mode as SilentLogin: some authentication mechanisms (like
// oauth2.TokenSource) require valid tokens no matter what. The corresponding
// factories set requiresAuth to true.
//
// Returns:
//   (true, nil) if successfully initialized the authenticator with some token.
//   (false, nil) to disable authentication (for OptionalLogin mode).
//   (false, err) on errors.
func (a *Authenticator) doLoginIfRequired(requiresAuth bool) (useAuth bool, err error) {
	if a.loginMode == InteractiveLogin {
		if err := a.PurgeCredentialsCache(); err != nil {
			return false, err
		}
	}
	effectiveMode := a.loginMode
	if requiresAuth && effectiveMode == OptionalLogin {
		effectiveMode = SilentLogin
	}
	switch err := a.CheckLoginRequired(); {
	case err == nil:
		return true, nil // have a valid cached base token
	case err == ErrInsufficientAccess && effectiveMode == OptionalLogin:
		return false, nil // have the base token, but it doesn't have enough scopes
	case err != ErrLoginRequired:
		return false, err // some error we can't handle (we handle only ErrLoginRequired)
	case effectiveMode == SilentLogin:
		return false, ErrLoginRequired // can't run Login in SilentLogin mode
	case effectiveMode == OptionalLogin:
		return false, nil // we can skip auth in OptionalLogin if we have no token
	case effectiveMode != InteractiveLogin:
		return false, fmt.Errorf("invalid mode argument: %s", effectiveMode)
	}
	if err := a.Login(); err != nil {
		return false, err
	}
	return true, nil
}

// currentToken returns currently loaded authentication token (or nil).
//
// It lock a.lock inside. It MUST NOT be called when a.lock is held. It will
// lazily call 'ensureInitialized' if necessary, returning its error.
func (a *Authenticator) currentToken() (tok *oauth2.Token, err error) {
	a.lock.RLock()
	initialized, err := a.checkInitialized()
	if initialized && err == nil {
		tok = a.authToken.token
	}
	a.lock.RUnlock()

	if !initialized {
		a.lock.Lock()
		defer a.lock.Unlock()
		if err = a.ensureInitialized(); err == nil {
			tok = a.authToken.token
		}
	}

	return
}

// refreshToken compares current auth token to 'prev' and launches token refresh
// procedure if they still match.
//
// Returns a refreshed token (if a refresh procedure happened) or the current
// token, if it's already different from 'prev'. Acts as "Compare-And-Swap"
// where "Swap" is a token refresh procedure.
//
// If the token can't be refreshed (e.g. the base token or the credentials were
// revoked), sets the current auth token to nil and returns an error.
func (a *Authenticator) refreshToken(prev *oauth2.Token, lifetime time.Duration) (*oauth2.Token, error) {
	return a.authToken.compareAndRefresh(a.ctx, compareAndRefreshOp{
		lock:     &a.lock,
		prev:     prev,
		lifetime: lifetime,
		refreshCb: func(ctx context.Context, prev *oauth2.Token) (*oauth2.Token, error) {
			// In Actor mode, need to make sure we have a sufficiently fresh base
			// token first, since it's needed to get the new auth token. 30 sec should
			// be more than enough to make an IAM call.
			var base *oauth2.Token
			if a.isActing() {
				var err error
				if base, err = a.getBaseTokenLocked(ctx, 30*time.Second); err != nil {
					return nil, err
				}
			}
			return a.authToken.renewToken(ctx, prev, base)
		},
	})
}

// getBaseTokenLocked is used to get an IAM-scoped token when running in
// actor mode.
//
// It is called with a.lock locked.
func (a *Authenticator) getBaseTokenLocked(ctx context.Context, lifetime time.Duration) (*oauth2.Token, error) {
	if !a.isActing() {
		panic("impossible")
	}

	// Already have a good token?
	if !internal.TokenExpiresInRnd(ctx, a.baseToken.token, lifetime) {
		return a.baseToken.token, nil
	}

	// Need to make one.
	return a.baseToken.compareAndRefresh(ctx, compareAndRefreshOp{
		lock:     nil, // already holding the lock
		prev:     a.baseToken.token,
		lifetime: lifetime,
		refreshCb: func(ctx context.Context, prev *oauth2.Token) (*oauth2.Token, error) {
			return a.baseToken.renewToken(ctx, prev, nil)
		},
	})
}

////////////////////////////////////////////////////////////////////////////////
// Transport implementation.

// authTokenInjector injects an authentication token into request headers.
//
// Used as a callback for NewModifyingTransport.
func (a *Authenticator) authTokenInjector(req *http.Request) error {
	// Grab a currently known token or fail if 'ensureInitialized' failed.
	tok, err := a.currentToken()
	if err != nil {
		return err
	}

	// Attempt to refresh the token, if required. Revert to non-authed call if
	// token can't be refreshed and running in OptionalLogin mode.
	if tok == nil || internal.TokenExpiresInRnd(a.ctx, tok, minAcceptedLifetime) {
		var err error
		tok, err = a.refreshToken(tok, minAcceptedLifetime)
		switch {
		case err == ErrLoginRequired && a.loginMode == OptionalLogin:
			return nil // skip auth, no need for modifications
		case err != nil:
			return err
		// Note: no randomization here. It is a sanity check that verifies
		// refreshToken did its job.
		case internal.TokenExpiresIn(a.ctx, tok, minAcceptedLifetime):
			return fmt.Errorf("auth: failed to refresh the token")
		}
	}
	tok.SetAuthHeader(req)
	return nil
}

////////////////////////////////////////////////////////////////////////////////
// tokenWithProvider implementation.

// tokenWithProvider wraps a token with provider that can update it and a cache
// that stores it.
type tokenWithProvider struct {
	token    *oauth2.Token          // in-memory cache of the token
	provider internal.TokenProvider // knows how to generate 'token'
	cache    internal.TokenCache    // persistent cache for the token
}

// fetchFromCache updates 't.token' by reading it from the cache.
func (t *tokenWithProvider) fetchFromCache(ctx context.Context) error {
	key, err := t.provider.CacheKey(ctx)
	if err != nil {
		return err
	}
	tok, err := t.cache.GetToken(key)
	if err != nil {
		return err
	}
	t.token = tok
	return nil
}

// putToCache puts 't.token' value into the cache.
func (t *tokenWithProvider) putToCache(ctx context.Context) error {
	key, err := t.provider.CacheKey(ctx)
	if err != nil {
		return err
	}
	return t.cache.PutToken(key, t.token)
}

// purgeToken removes the token from both on-disk cache and memory.
func (t *tokenWithProvider) purgeToken(ctx context.Context) error {
	t.token = nil
	key, err := t.provider.CacheKey(ctx)
	if err != nil {
		return err
	}
	return t.cache.DeleteToken(key)
}

// compareAndRefreshOp is parameters for 'compareAndRefresh' call.
type compareAndRefreshOp struct {
	lock      sync.Locker   // optional lock to grab when comparing and refreshing
	prev      *oauth2.Token // previously known token (the one we are refreshing)
	lifetime  time.Duration // minimum acceptable token lifetime
	refreshCb func(ctx context.Context, existing *oauth2.Token) (*oauth2.Token, error)
}

// compareAndRefresh compares currently stored token to 'prev' and calls the
// given callback (under the lock, if not nil) to refresh it if they are still
// equal.
//
// Returns a refreshed token (if a refresh procedure happened) or the current
// token, if it's already different from 'prev'. Acts as "Compare-And-Swap"
// where "Swap" is a token refresh callback.
//
// If the callback returns an error (meaning the token can't be refreshed), sets
// the token to nil and returns the error.
func (t *tokenWithProvider) compareAndRefresh(ctx context.Context, params compareAndRefreshOp) (*oauth2.Token, error) {
	cacheKey, err := t.provider.CacheKey(ctx)
	if err != nil {
		// An error here is truly fatal. It is something like "can't read service
		// account JSON from disk". There's no way to refresh a token without it.
		return nil, err
	}

	// To give a context to "Minting a new token" messages and similar below.
	ctx = logging.SetFields(ctx, logging.Fields{
		"key":    cacheKey.Key,
		"scopes": strings.Join(cacheKey.Scopes, " "),
	})

	// Check that the token still need a refresh and do it (under the lock).
	tok, cacheIt, err := func() (*oauth2.Token, bool, error) {
		if params.lock != nil {
			params.lock.Lock()
			defer params.lock.Unlock()
		}

		// Some other goroutine already updated the token, just return the new one.
		if t.token != nil && !internal.EqualTokens(t.token, params.prev) {
			return t.token, false, nil
		}

		// Rescan the cache. Maybe some other process updated the token. This branch
		// is also responsible for lazy-loading of tokens from cache for
		// non-interactive providers, see ensureInitialized().
		if cached, _ := t.cache.GetToken(cacheKey); cached != nil {
			t.token = cached
			if !internal.EqualTokens(cached, params.prev) && !internal.TokenExpiresIn(ctx, cached, params.lifetime) {
				return cached, false, nil
			}
		}

		// No one updated the token yet. It should be us. Mint a new token or
		// refresh the existing one.
		newTok, err := params.refreshCb(ctx, t.token)
		if err != nil {
			t.token = nil
			return nil, false, err
		}
		logging.Debugf(ctx, "Token expires in %s", newTok.Expiry.Sub(clock.Now(ctx)))
		t.token = newTok
		return newTok, true, nil
	}()

	if err == internal.ErrBadRefreshToken || err == internal.ErrBadCredentials {
		// Do not keep the broken token in the cache. It is unusable. Do this
		// outside the lock to avoid blocking other callers. Note that t.token is
		// already nil.
		if err := t.cache.DeleteToken(cacheKey); err != nil {
			logging.Warningf(ctx, "Failed to remove broken token from the cache: %s", err)
		}
		// A bad refresh token can be fixed by interactive login, so adjust the
		// error accordingly in this case.
		if err == internal.ErrBadRefreshToken {
			err = ErrLoginRequired
		}
	}

	if err != nil {
		return nil, err
	}

	// Update the cache outside the lock, no need for callers to wait for this.
	// Do not die if failed, the token is still usable from the memory.
	if cacheIt && tok != nil {
		if err := t.cache.PutToken(cacheKey, tok); err != nil {
			logging.Warningf(ctx, "Failed to write refreshed token to the cache: %s", err)
		}
	}

	return tok, nil
}

// renewToken is called to mint a new token or update existing one.
//
// It is called from non-interactive 'refreshToken' method, and thus it can't
// use interactive login flow.
func (t *tokenWithProvider) renewToken(ctx context.Context, prev, base *oauth2.Token) (*oauth2.Token, error) {
	if prev == nil {
		if t.provider.RequiresInteraction() {
			return nil, ErrLoginRequired
		}
		logging.Debugf(ctx, "Minting a new token")
		tok, err := t.mintTokenWithRetries(ctx, base)
		if err != nil {
			logging.Warningf(ctx, "Failed to mint a token: %s", err)
			return nil, err
		}
		return tok, nil
	}

	logging.Debugf(ctx, "Refreshing the token")
	tok, err := t.refreshTokenWithRetries(ctx, prev, base)
	if err != nil {
		logging.Warningf(ctx, "Failed to refresh the token: %s", err)
		return nil, err
	}
	return tok, nil
}

// retryParams defines retry strategy for handling transient errors when minting
// or refreshing tokens.
func retryParams() retry.Iterator {
	return &retry.ExponentialBackoff{
		Limited: retry.Limited{
			Delay:    50 * time.Millisecond,
			Retries:  50,
			MaxTotal: 5 * time.Second,
		},
		Multiplier: 2,
	}
}

// mintTokenWithRetries calls provider's MintToken() retrying on transient
// errors a bunch of times. Called only for non-interactive providers.
func (t *tokenWithProvider) mintTokenWithRetries(ctx context.Context, base *oauth2.Token) (tok *oauth2.Token, err error) {
	err = retry.Retry(ctx, retry.TransientOnly(retryParams), func() error {
		tok, err = t.provider.MintToken(ctx, base)
		return err
	}, nil)
	return
}

// refreshTokenWithRetries calls providers' RefreshToken(...) retrying on
// transient errors a bunch of times.
func (t *tokenWithProvider) refreshTokenWithRetries(ctx context.Context, prev, base *oauth2.Token) (tok *oauth2.Token, err error) {
	err = retry.Retry(ctx, retry.TransientOnly(retryParams), func() error {
		tok, err = t.provider.RefreshToken(ctx, prev, base)
		return err
	}, nil)
	return
}

////////////////////////////////////////////////////////////////////////////////
// Utility functions.

// makeBaseTokenProvider creates TokenProvider implementation based on options.
//
// opts.Scopes is ignored, 'scopes' is used instead. This is used in actor mode
// to substitute scopes with IAM scope.
//
// Called by ensureInitialized.
func makeBaseTokenProvider(ctx context.Context, opts *Options, scopes []string) (internal.TokenProvider, error) {
	if opts.testingBaseTokenProvider != nil {
		return opts.testingBaseTokenProvider, nil
	}

	switch opts.Method {
	case UserCredentialsMethod:
		return internal.NewUserAuthTokenProvider(
			ctx,
			opts.ClientID,
			opts.ClientSecret,
			scopes)
	case ServiceAccountMethod:
		serviceAccountPath := ""
		if len(opts.ServiceAccountJSON) == 0 {
			serviceAccountPath = opts.ServiceAccountJSONPath
		}
		return internal.NewServiceAccountTokenProvider(
			ctx,
			opts.ServiceAccountJSON,
			serviceAccountPath,
			scopes)
	case GCEMetadataMethod:
		return internal.NewGCETokenProvider(ctx, opts.GCEAccountName, scopes)
	case LUCIContextMethod:
		return internal.NewLUCIContextTokenProvider(ctx, scopes, opts.Transport)
	default:
		return nil, fmt.Errorf("auth: unrecognized authentication method: %s", opts.Method)
	}
}

// makeIAMTokenProvider create TokenProvider to use in Actor mode.
//
// Called by ensureInitialized in actor mode.
func makeIAMTokenProvider(ctx context.Context, opts *Options) (internal.TokenProvider, error) {
	if opts.testingIAMTokenProvider != nil {
		return opts.testingIAMTokenProvider, nil
	}
	return internal.NewIAMTokenProvider(
		ctx,
		opts.ActAsServiceAccount,
		opts.Scopes,
		opts.Transport)
}
