// Code generated by github.com/UnAfraid/dataloaden, DO NOT EDIT.

package dataloader

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/UnAfraid/dataloaden/v2/dataloader"
	"github.com/UnAfraid/wg-ui/api/model"
	"github.com/hashicorp/go-multierror"
)

// NewServerLoader creates a new ServerLoader given a fetch, wait, and maxBatch
func NewServerLoader(config dataloader.Config[string, *model.Server]) dataloader.DataLoader[string, *model.Server] {
	dl := &ServerLoader{
		fetch:        config.Fetch,
		wait:         config.Wait,
		formatErrors: config.FormatErrors,
		maxBatch:     config.MaxBatch,
	}
	if dl.formatErrors == nil {
		dl.formatErrors = dl.defaultFormatErrors
	}
	return dl
}

// ServerLoader batches and caches requests
type ServerLoader struct {
	// this method provides the data for the loader
	fetch func(keys []string) ([]*model.Server, []error)

	// how long to done before sending a batch
	wait time.Duration

	// this will limit the maximum number of keys to send in one batch, 0 = no limit
	maxBatch int

	// this method will format errors
	formatErrors func([]error) string

	// INTERNAL

	// lazily created cache
	cache map[string]*model.Server

	// the current batch. keys will continue to be collected until timeout is hit,
	// then everything will be sent to the fetch method and out to the listeners
	batch *serverLoaderBatch

	// mutex to prevent races
	mu sync.Mutex
}

type serverLoaderBatch struct {
	keys    []string
	data    []*model.Server
	error   []error
	closing bool
	done    chan struct{}
}

// Load a Server by key, batching and caching will be applied automatically
func (l *ServerLoader) Load(key string) (*model.Server, error) {
	return l.LoadThunk(key)()
}

// LoadThunk returns a function that when called will block waiting for a Server.
// This method should be used if you want one goroutine to make requests to many
// different data loaders without blocking until the thunk is called.
func (l *ServerLoader) LoadThunk(key string) func() (*model.Server, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if it, ok := l.cache[key]; ok {
		return func() (*model.Server, error) {
			return it, nil
		}
	}
	if l.batch == nil {
		l.batch = &serverLoaderBatch{done: make(chan struct{})}
	}
	batch := l.batch
	pos := batch.keyIndex(l, key)

	return func() (*model.Server, error) {
		<-batch.done

		var data *model.Server
		if pos < len(batch.data) {
			data = batch.data[pos]
		}

		var errs error
		for _, err := range batch.error {
			if err == nil {
				continue
			}
			if errs == nil {
				errs = err
			} else {
				errs = multierror.Append(errs, err)
			}
		}
		if errs != nil {
			if multiErr, ok := errs.(*multierror.Error); ok {
				multiErr.ErrorFormat = l.formatErrors
			}
			return data, errs
		}

		l.mu.Lock()
		defer l.mu.Unlock()
		l.unsafeSet(key, data)

		return data, nil
	}
}

// LoadAll fetches many keys at once. It will be broken into appropriate sized
// sub batches depending on how the loader is configured
func (l *ServerLoader) LoadAll(keys []string) ([]*model.Server, []error) {
	results := make([]func() (*model.Server, error), len(keys))

	for i, key := range keys {
		results[i] = l.LoadThunk(key)
	}

	servers := make([]*model.Server, len(keys))
	errors := make([]error, len(keys))
	for i, thunk := range results {
		servers[i], errors[i] = thunk()
	}
	return servers, errors
}

// LoadAllThunk returns a function that when called will block waiting for a Servers.
// This method should be used if you want one goroutine to make requests to many
// different data loaders without blocking until the thunk is called.
func (l *ServerLoader) LoadAllThunk(keys []string) func() ([]*model.Server, []error) {
	results := make([]func() (*model.Server, error), len(keys))
	for i, key := range keys {
		results[i] = l.LoadThunk(key)
	}
	return func() ([]*model.Server, []error) {
		servers := make([]*model.Server, len(keys))
		errors := make([]error, len(keys))
		for i, thunk := range results {
			servers[i], errors[i] = thunk()
		}
		return servers, errors
	}
}

// Prime the cache with the provided key and value. If the key already exists, no change is made
// and false is returned.
// (To forcefully prime the cache, clear the key first with loader.clear(key).prime(key, value).)
func (l *ServerLoader) Prime(key string, value *model.Server) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	var found bool
	if _, found = l.cache[key]; !found {
		// make a copy when writing to the cache, its easy to pass a pointer in from a loop var
		// and end up with the whole cache pointing to the same value.
		cpy := *value
		l.unsafeSet(key, &cpy)
	}
	return !found
}

// Clear the value at key from the cache, if it exists
func (l *ServerLoader) Clear(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.cache, key)
}

// defaultFormatErrors would format multiple errors
func (l *ServerLoader) defaultFormatErrors(errors []error) string {
	if len(errors) == 1 {
		return errors[0].Error()
	}

	countsByErrors := make(map[string]int)
	for _, err := range errors {
		countsByErrors[err.Error()]++
	}

	type errorOccurrences struct {
		error       string
		occurrences int
	}

	var sortedErrorOccurrences []errorOccurrences
	for err, count := range countsByErrors {
		sortedErrorOccurrences = append(sortedErrorOccurrences, errorOccurrences{
			error:       err,
			occurrences: count,
		})
	}

	sort.Slice(sortedErrorOccurrences, func(i, j int) bool {
		return sortedErrorOccurrences[i].occurrences > sortedErrorOccurrences[j].occurrences
	})

	var sb strings.Builder
	for _, seo := range sortedErrorOccurrences {
		sb.WriteString(" * ")
		sb.WriteString(strconv.Itoa(seo.occurrences))
		sb.WriteString(" ")
		sb.WriteString(seo.error)
		sb.WriteString("\n")
	}

	return fmt.Sprintf("%d errors occurred:\n%s\n", len(errors), sb.String())
}

func (l *ServerLoader) unsafeSet(key string, value *model.Server) {
	if l.cache == nil {
		l.cache = map[string]*model.Server{}
	}
	l.cache[key] = value
}

// keyIndex will return the location of the key in the batch, if its not found
// it will add the key to the batch
func (b *serverLoaderBatch) keyIndex(l *ServerLoader, key string) int {
	for i, existingKey := range b.keys {
		if key == existingKey {
			return i
		}
	}

	pos := len(b.keys)
	b.keys = append(b.keys, key)
	if pos == 0 {
		go b.startTimer(l)
	}

	if l.maxBatch != 0 && pos >= l.maxBatch-1 {
		if !b.closing {
			b.closing = true
			l.batch = nil
			go b.end(l)
		}
	}

	return pos
}

func (b *serverLoaderBatch) startTimer(l *ServerLoader) {
	time.Sleep(l.wait)
	l.mu.Lock()
	defer l.mu.Unlock()

	// we must have hit a batch limit and are already finalizing this batch
	if b.closing {
		return
	}

	l.batch = nil
	b.end(l)
}

func (b *serverLoaderBatch) end(l *ServerLoader) {
	b.data, b.error = l.fetch(b.keys)
	close(b.done)
}
