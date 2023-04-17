package dataloader

import (
	"time"
)

// Config captures the config to create a new DataLoader
type Config[K, V any] struct {
	// Fetch is a method that provides the data for the loader
	Fetch func(keys []K) ([]V, []error)

	// Wait is how long wait before sending a batch
	Wait time.Duration

	// FormatErrors will format multiple errors as one
	FormatErrors func([]error) string

	// MaxBatch will limit the maximum number of keys to send in one batch, 0 = not limit
	MaxBatch int
}
