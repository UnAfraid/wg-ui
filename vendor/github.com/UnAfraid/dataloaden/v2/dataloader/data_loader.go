package dataloader

// DataLoader batches and caches requests
type DataLoader[K, V any] interface {
	// Load a User by key, batching and caching will be applied automatically
	Load(key K) (V, error)

	// LoadThunk returns a function that when called will block waiting for a User.
	// This method should be used if you want one goroutine to make requests to many
	// different data loaders without blocking until the thunk is called.
	LoadThunk(key K) func() (V, error)

	// LoadAll fetches many keys at once. It will be broken into appropriate sized
	// sub batches depending on how the loader is configured
	LoadAll(keys []K) ([]V, []error)

	// LoadAllThunk returns a function that when called will block waiting for a Users.
	// This method should be used if you want one goroutine to make requests to many
	// different data loaders without blocking until the thunk is called.
	LoadAllThunk(keys []K) func() ([]V, []error)

	// Prime the cache with the provided key and value. If the key already exists, no change is made
	// and false is returned.
	// (To forcefully prime the cache, clear the key first with loader.clear(key).prime(key, value).)
	Prime(key K, value V) bool

	// Clear the value at key from the cache, if it exists
	Clear(key K)
}
