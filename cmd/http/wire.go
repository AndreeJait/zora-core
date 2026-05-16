package main

import (
	"context"
	"net/http"

	"github.com/AndreeJait/zora-core/config"
	"go.uber.org/dig"
)

// CleanupCollector accumulates cleanup functions from infrastructure providers.
// After dig resolves all dependencies, its Cleanup method is passed to gracefulw.
type CleanupCollector struct {
	cleanups []func(ctx context.Context) error
}

// Add appends a cleanup function.
func (cc *CleanupCollector) Add(fn func(ctx context.Context) error) {
	cc.cleanups = append(cc.cleanups, fn)
}

// Cleanup runs all collected cleanup functions, returning the first error.
func (cc *CleanupCollector) Cleanup(ctx context.Context) error {
	var firstErr error
	for _, fn := range cc.cleanups {
		if err := fn(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// wire builds the dependency graph using dig and returns the HTTP handler + cleanup.
// The dig container is stored in the package-level digContainer variable for
// post-wire extraction by main.go.
var digContainer *dig.Container

func wire(cfg *config.AppConfig) (http.Handler, func(ctx context.Context) error, error) {
	c := dig.New()
	digContainer = c

	// Root values
	c.Provide(func() *config.AppConfig { return cfg })
	c.Provide(func() *CleanupCollector { return &CleanupCollector{} })

	// Register all providers
	provideInfrastructure(c)
	provideServices(c)
	provideRouter(c)

	// Invoke to build the handler
	var handler http.Handler
	if err := c.Invoke(func(h http.Handler) {
		handler = h
	}); err != nil {
		return nil, nil, err
	}

	// Retrieve cleanup collector
	var cc *CleanupCollector
	if err := c.Invoke(func(collector *CleanupCollector) {
		cc = collector
	}); err != nil {
		return nil, nil, err
	}

	return handler, cc.Cleanup, nil
}