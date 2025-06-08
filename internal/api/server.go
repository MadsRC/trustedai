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

type Server struct {
	options *serverOptions
	mux     *http.ServeMux
}

// NewServer creates a new [Server].
func NewServer(options ...ServerOption) (*Server, error) {
	opts := defaultServerOptions
	for _, opt := range GlobalServerOptions {
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

	return &Server{
		options: &opts,
		mux:     mux,
	}, nil
}

type serverOptions struct {
	Logger                 *slog.Logger
	UserRepository         llmgw.UserRepository
	OrganizationRepository llmgw.OrganizationRepository
	TokenRepository        llmgw.TokenRepository
	SsoHandler             http.Handler
	SessionStore           auth.SessionStore
	AuthInterceptor        *auth.Interceptor
	TokenInterceptor       *auth.TokenInterceptor
}

var defaultServerOptions = serverOptions{
	Logger: slog.Default(),
}

// GlobalServerOptions is a list of [ServerOption]s that are applied to all [Server]s.
var GlobalServerOptions []ServerOption

// ServerOption is an option for configuring a [Server].
type ServerOption interface {
	apply(*serverOptions)
}

// funcServerOption is a [ServerOption] that calls a function.
// It is used to wrap a function, so it satisfies the [ServerOption] interface.
type funcServerOption struct {
	f func(*serverOptions)
}

func (fdo *funcServerOption) apply(opts *serverOptions) {
	fdo.f(opts)
}

func newFuncServerOption(f func(*serverOptions)) *funcServerOption {
	return &funcServerOption{
		f: f,
	}
}

// WithServerLogger returns a [ServerOption] that uses the provided logger.
func WithServerLogger(logger *slog.Logger) ServerOption {
	return newFuncServerOption(func(opts *serverOptions) {
		opts.Logger = logger
	})
}

// WithServerUserRepository returns a [ServerOption] that uses the provided UserRepository.
func WithServerUserRepository(repository llmgw.UserRepository) ServerOption {
	return newFuncServerOption(func(opts *serverOptions) {
		opts.UserRepository = repository
	})
}

// WithServerOrganizationRepository returns a [ServerOption] that uses the provided OrganizationRepository.
func WithServerOrganizationRepository(repository llmgw.OrganizationRepository) ServerOption {
	return newFuncServerOption(func(opts *serverOptions) {
		opts.OrganizationRepository = repository
	})
}

// WithServerTokenRepository returns a [ServerOption] that uses the provided TokenRepository.
func WithServerTokenRepository(repository llmgw.TokenRepository) ServerOption {
	return newFuncServerOption(func(opts *serverOptions) {
		opts.TokenRepository = repository
	})
}

// WithSSOHandler returns a [ServerOption] that uses the provided SSO handler.
func WithSSOHandler(handler http.Handler) ServerOption {
	return newFuncServerOption(func(opts *serverOptions) {
		opts.SsoHandler = handler
	})
}

// WithSessionStore returns a [ServerOption] that uses the provided session store.
func WithSessionStore(store auth.SessionStore) ServerOption {
	return newFuncServerOption(func(opts *serverOptions) {
		opts.SessionStore = store
	})
}

// WithAuthInterceptor returns a [ServerOption] that uses the provided auth interceptor.
func WithAuthInterceptor(interceptor *auth.Interceptor) ServerOption {
	return newFuncServerOption(func(opts *serverOptions) {
		opts.AuthInterceptor = interceptor
	})
}

// WithTokenInterceptor returns a [ServerOption] that uses the provided token interceptor.
func WithTokenInterceptor(interceptor *auth.TokenInterceptor) ServerOption {
	return newFuncServerOption(func(opts *serverOptions) {
		opts.TokenInterceptor = interceptor
	})
}

// GetMux returns the HTTP mux for the server
func (s *Server) GetMux() *http.ServeMux {
	return s.mux
}

func (s *Server) Run() {
	fmt.Println("Starting server on localhost:9999")
	_ = http.ListenAndServe(
		"localhost:9999",
		// Use h2c so we can serve HTTP/2 without TLS.
		h2c.NewHandler(s.mux, &http2.Server{}),
	)
}
