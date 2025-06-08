// SPDX-FileCopyrightText: 2025 Mads R. Havmand <mads@v42.dk>
//
// SPDX-License-Identifier: AGPL-3.0-only

package api

import (
	"fmt"
	"log/slog"
	"net/http"

	"codeberg.org/MadsRC/llmgw"
	"codeberg.org/MadsRC/llmgw/gen/proto/madsrc/llmgw/v1/llmgwv1connect"
	"codeberg.org/MadsRC/llmgw/internal/api/auth"
	"codeberg.org/MadsRC/llmgw/internal/api/services"
	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func registerServiceHandlers(
	mux *http.ServeMux,
	interceptors []connect.Interceptor,
	servicesToRegister map[string]any,
) {
	for path, handler := range servicesToRegister {
		mux.Handle(path, handler.(http.Handler))
	}
}

type ControlPlaneServer struct {
	options *controlPlaneOptions
	mux     *http.ServeMux
}

// NewControlPlaneServer creates a new [ControlPlaneServer].
func NewControlPlaneServer(options ...ControlPlaneOption) (*ControlPlaneServer, error) {
	opts := defaultControlPlaneOptions
	for _, opt := range GlobalControlPlaneOptions {
		opt.apply(&opts)
	}
	for _, opt := range options {
		opt.apply(&opts)
	}

	// Create auth interceptor if not provided
	authInterceptor := opts.AuthInterceptor
	if authInterceptor == nil && opts.SessionStore != nil {
		authInterceptor = auth.NewInterceptor(opts.SessionStore)
	}

	// Create IAM service handler
	iamServiceHandler, err := services.NewIam(
		services.WithUserRepository(opts.UserRepository),
		services.WithOrganizationRepository(opts.OrganizationRepository),
		services.WithTokenRepository(opts.TokenRepository),
	)
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()

	// Create handler with interceptors
	interceptors := []connect.Interceptor{}
	if authInterceptor != nil {
		interceptors = append(interceptors, authInterceptor)
	}
	if opts.TokenInterceptor != nil {
		interceptors = append(interceptors, opts.TokenInterceptor)
	}

	servicesToRegister := make(map[string]any)

	// IAM service
	iamPath, iamHandler := llmgwv1connect.NewIAMServiceHandler(
		iamServiceHandler,
		connect.WithInterceptors(interceptors...),
	)
	servicesToRegister[iamPath] = iamHandler

	registerServiceHandlers(mux, interceptors, servicesToRegister)

	// Add gRPC reflection support
	reflector := grpcreflect.NewStaticReflector(
		llmgwv1connect.IAMServiceName,
	)
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	// Add SSO callback handler if provided
	if opts.SsoHandler != nil {
		mux.Handle("/sso/", http.StripPrefix("/sso", opts.SsoHandler))
	}

	return &ControlPlaneServer{
		options: &opts,
		mux:     mux,
	}, nil
}

type controlPlaneOptions struct {
	Logger                 *slog.Logger
	UserRepository         llmgw.UserRepository
	OrganizationRepository llmgw.OrganizationRepository
	TokenRepository        llmgw.TokenRepository
	SsoHandler             http.Handler
	SessionStore           auth.SessionStore
	AuthInterceptor        *auth.Interceptor
	TokenInterceptor       *auth.TokenInterceptor
}

var defaultControlPlaneOptions = controlPlaneOptions{
	Logger: slog.Default(),
}

// GlobalControlPlaneOptions is a list of [ControlPlaneOption]s that are applied to all [ControlPlaneServer]s.
var GlobalControlPlaneOptions []ControlPlaneOption

// ControlPlaneOption is an option for configuring a [ControlPlaneServer].
type ControlPlaneOption interface {
	apply(*controlPlaneOptions)
}

// funcControlPlaneOption is a [ControlPlaneOption] that calls a function.
// It is used to wrap a function, so it satisfies the [ControlPlaneOption] interface.
type funcControlPlaneOption struct {
	f func(*controlPlaneOptions)
}

func (fdo *funcControlPlaneOption) apply(opts *controlPlaneOptions) {
	fdo.f(opts)
}

func newFuncControlPlaneOption(f func(*controlPlaneOptions)) *funcControlPlaneOption {
	return &funcControlPlaneOption{
		f: f,
	}
}

// WithControlPlaneLogger returns a [ControlPlaneOption] that uses the provided logger.
func WithControlPlaneLogger(logger *slog.Logger) ControlPlaneOption {
	return newFuncControlPlaneOption(func(opts *controlPlaneOptions) {
		opts.Logger = logger
	})
}

// WithControlPlaneUserRepository returns a [ControlPlaneOption] that uses the provided UserRepository.
func WithControlPlaneUserRepository(repository llmgw.UserRepository) ControlPlaneOption {
	return newFuncControlPlaneOption(func(opts *controlPlaneOptions) {
		opts.UserRepository = repository
	})
}

// WithControlPlaneOrganizationRepository returns a [ControlPlaneOption] that uses the provided OrganizationRepository.
func WithControlPlaneOrganizationRepository(repository llmgw.OrganizationRepository) ControlPlaneOption {
	return newFuncControlPlaneOption(func(opts *controlPlaneOptions) {
		opts.OrganizationRepository = repository
	})
}

// WithControlPlaneTokenRepository returns a [ControlPlaneOption] that uses the provided TokenRepository.
func WithControlPlaneTokenRepository(repository llmgw.TokenRepository) ControlPlaneOption {
	return newFuncControlPlaneOption(func(opts *controlPlaneOptions) {
		opts.TokenRepository = repository
	})
}

// WithSSOHandler returns a [ControlPlaneOption] that uses the provided SSO handler.
func WithSSOHandler(handler http.Handler) ControlPlaneOption {
	return newFuncControlPlaneOption(func(opts *controlPlaneOptions) {
		opts.SsoHandler = handler
	})
}

// WithSessionStore returns a [ControlPlaneOption] that uses the provided session store.
func WithSessionStore(store auth.SessionStore) ControlPlaneOption {
	return newFuncControlPlaneOption(func(opts *controlPlaneOptions) {
		opts.SessionStore = store
	})
}

// WithAuthInterceptor returns a [ControlPlaneOption] that uses the provided auth interceptor.
func WithAuthInterceptor(interceptor *auth.Interceptor) ControlPlaneOption {
	return newFuncControlPlaneOption(func(opts *controlPlaneOptions) {
		opts.AuthInterceptor = interceptor
	})
}

// WithTokenInterceptor returns a [ControlPlaneOption] that uses the provided token interceptor.
func WithTokenInterceptor(interceptor *auth.TokenInterceptor) ControlPlaneOption {
	return newFuncControlPlaneOption(func(opts *controlPlaneOptions) {
		opts.TokenInterceptor = interceptor
	})
}

// GetMux returns the HTTP mux for the server
func (s *ControlPlaneServer) GetMux() *http.ServeMux {
	return s.mux
}

func (s *ControlPlaneServer) Run() {
	fmt.Println("Starting server on localhost:9999")
	_ = http.ListenAndServe(
		"localhost:9999",
		// Use h2c so we can serve HTTP/2 without TLS.
		h2c.NewHandler(s.mux, &http2.Server{}),
	)
}
