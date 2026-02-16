package bbolt

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/UnAfraid/searchindex"
	"go.etcd.io/bbolt"

	"github.com/UnAfraid/wg-ui/pkg/backend"
	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
)

const (
	backendBucket = "backend"
)

type backendRepository struct {
	db *bbolt.DB
}

func NewBackendRepository(db *bbolt.DB) backend.Repository {
	return &backendRepository{
		db: db,
	}
}

func (r *backendRepository) FindOne(ctx context.Context, options *backend.FindOneOptions) (*backend.Backend, error) {
	return dbTx(ctx, r.db, backendBucket, false, func(tx *bbolt.Tx, bucket *bbolt.Bucket) (*backend.Backend, error) {
		if idOption := options.IdOption; idOption != nil {
			jsonState := bucket.Get([]byte(idOption.Id))
			if jsonState == nil {
				return nil, nil
			}

			var b *backend.Backend
			if err := json.Unmarshal(jsonState, &b); err != nil {
				return nil, fmt.Errorf("failed to unmarshal backend: %w", err)
			}

			return b, nil
		} else if nameOption := options.NameOption; nameOption != nil {
			var b *backend.Backend
			c := bucket.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				if err := json.Unmarshal(v, &b); err != nil {
					return nil, fmt.Errorf("failed to unmarshal backend: %w", err)
				}
				if strings.EqualFold(b.Name, nameOption.Name) {
					return b, nil
				}
			}
		}

		return nil, nil
	})
}

func (r *backendRepository) FindAll(ctx context.Context, options *backend.FindOptions) ([]*backend.Backend, error) {
	return dbTx(ctx, r.db, backendBucket, false, func(tx *bbolt.Tx, bucket *bbolt.Bucket) ([]*backend.Backend, error) {
		var backends []*backend.Backend
		var backendsCount int
		var searchList searchindex.SearchList[*backend.Backend]
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			backendsCount++

			var b *backend.Backend
			if err := json.Unmarshal(v, &b); err != nil {
				return nil, fmt.Errorf("failed to unmarshal backend: %w", err)
			}

			var optionsLen int
			if len(options.Ids) != 0 {
				optionsLen++
				if slices.Contains(options.Ids, b.Id) {
					backends = append(backends, b)
					continue
				}
			}

			if options.Type != nil {
				optionsLen++
				if b.Type() == *options.Type {
					backends = append(backends, b)
					continue
				}
			}

			if options.Enabled != nil {
				optionsLen++
				if b.Enabled == *options.Enabled {
					backends = append(backends, b)
					continue
				}
			}

			if options.CreateUserId != nil {
				optionsLen++
				if b.CreateUserId == *options.CreateUserId {
					backends = append(backends, b)
					continue
				}
			}

			if options.UpdateUserId != nil {
				optionsLen++
				if b.UpdateUserId == *options.UpdateUserId {
					backends = append(backends, b)
					continue
				}
			}

			if len(options.Query) != 0 {
				optionsLen++
				searchList = append(searchList, &searchindex.SearchItem[*backend.Backend]{
					Key:  b.Name,
					Data: b,
				})
				searchList = append(searchList, &searchindex.SearchItem[*backend.Backend]{
					Key:  b.Description,
					Data: b,
				})
			}

			if optionsLen == 0 {
				backends = append(backends, b)
			}
		}

		if len(options.Query) != 0 {
			searchIndex := searchindex.NewSearchIndex(searchList, backendsCount, nil, nil, true, nil)
			matches := searchIndex.Search(searchindex.SearchParams[*backend.Backend]{
				Text:       options.Query,
				OutputSize: backendsCount,
				Matching:   searchindex.Beginning,
			})
			backends = append(backends, matches...)
		}

		return backends, nil
	})
}

func (r *backendRepository) Create(ctx context.Context, b *backend.Backend) (*backend.Backend, error) {
	return dbTx(ctx, r.db, backendBucket, true, func(tx *bbolt.Tx, bucket *bbolt.Bucket) (*backend.Backend, error) {
		id := []byte(b.Id)
		if bucket.Get(id) != nil {
			return nil, backend.ErrBackendIdAlreadyExists
		}

		jsonState, err := json.Marshal(b)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal backend: %w", err)
		}

		return b, bucket.Put(id, jsonState)
	})
}

func (r *backendRepository) Update(ctx context.Context, b *backend.Backend, fieldMask *backend.UpdateFieldMask) (*backend.Backend, error) {
	return dbTx(ctx, r.db, backendBucket, false, func(tx *bbolt.Tx, bucket *bbolt.Bucket) (*backend.Backend, error) {
		id := []byte(b.Id)
		jsonState := bucket.Get(id)
		if jsonState == nil {
			return nil, backend.ErrBackendNotFound
		}

		var updatedBackend *backend.Backend
		if err := json.Unmarshal(jsonState, &updatedBackend); err != nil {
			return nil, fmt.Errorf("failed to unmarshal backend: %w", err)
		}

		if fieldMask.Name {
			updatedBackend.Name = b.Name
		}

		if fieldMask.Description {
			updatedBackend.Description = b.Description
		}

		if fieldMask.Url {
			updatedBackend.Url = b.Url
		}

		if fieldMask.Enabled {
			updatedBackend.Enabled = b.Enabled
		}

		if fieldMask.UpdateUserId {
			updatedBackend.UpdateUserId = b.UpdateUserId
		}

		updatedBackend.UpdatedAt = time.Now()

		jsonState, err := json.Marshal(updatedBackend)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal backend: %w", err)
		}

		return updatedBackend, bucket.Put(id, jsonState)
	})
}

func (r *backendRepository) Delete(ctx context.Context, backendId string, deleteUserId string) (*backend.Backend, error) {
	return dbTx(ctx, r.db, backendBucket, false, func(tx *bbolt.Tx, bucket *bbolt.Bucket) (*backend.Backend, error) {
		id := []byte(backendId)
		jsonState := bucket.Get(id)
		if jsonState == nil {
			return nil, backend.ErrBackendNotFound
		}

		var deletedBackend *backend.Backend
		if err := json.Unmarshal(jsonState, &deletedBackend); err != nil {
			return nil, fmt.Errorf("failed to unmarshal backend: %w", err)
		}

		deletedBackend.DeleteUserId = deleteUserId
		deletedBackend.DeletedAt = adapt.ToPointer(time.Now())

		return deletedBackend, bucket.Delete(id)
	})
}
