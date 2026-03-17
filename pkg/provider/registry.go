package provider

import (
	"fmt"
	"sort"

	"github.com/dplense/dplense-cli/internal/logger"
	"github.com/dplense/dplense-cli/pkg/config"
)

// Factory creates a Provider from config and logger.
type Factory func(cfg *config.Config, log logger.Logger) (Provider, error)

var registry = make(map[Name]Factory)

// Register adds a provider factory to the registry.
// Typically called from a provider package's init() function.
func Register(name Name, factory Factory) {
	registry[name] = factory
}

// Get creates and returns a provider by name.
func Get(name Name, cfg *config.Config, log logger.Logger) (Provider, error) {
	factory, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %q (available: %v)", name, Available())
	}
	return factory(cfg, log)
}

// Available returns all registered provider names, sorted.
func Available() []Name {
	names := make([]Name, 0, len(registry))
	for k := range registry {
		names = append(names, k)
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })
	return names
}
