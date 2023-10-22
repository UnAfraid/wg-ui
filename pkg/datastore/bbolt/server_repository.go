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

	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
	"github.com/UnAfraid/wg-ui/pkg/server"
)

const (
	serverBucket = "server"
)

type serverRepository struct {
	db *bbolt.DB
}

func NewServerRepository(db *bbolt.DB) server.Repository {
	return &serverRepository{
		db: db,
	}
}

func (r *serverRepository) FindOne(ctx context.Context, options *server.FindOneOptions) (*server.Server, error) {
	return dbTx(ctx, r.db, serverBucket, false, func(tx *bbolt.Tx, bucket *bbolt.Bucket) (*server.Server, error) {
		if idOption := options.IdOption; idOption != nil {
			jsonState := bucket.Get([]byte(idOption.Id))
			if jsonState == nil {
				return nil, nil
			}

			var s *server.Server
			if err := json.Unmarshal(jsonState, &s); err != nil {
				return nil, fmt.Errorf("failed to unmarshal server: %w", err)
			}

			return s, nil
		} else if nameOption := options.NameOption; nameOption != nil {
			var s *server.Server
			c := bucket.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				if err := json.Unmarshal(v, &s); err != nil {
					return nil, fmt.Errorf("failed to unmarshal server: %w", err)
				}
				if strings.EqualFold(s.Name, nameOption.Name) {
					return s, nil
				}
			}
		}

		return nil, nil
	})
}

func (r *serverRepository) FindAll(ctx context.Context, options *server.FindOptions) ([]*server.Server, error) {
	return dbTx(ctx, r.db, serverBucket, false, func(tx *bbolt.Tx, bucket *bbolt.Bucket) ([]*server.Server, error) {
		var servers []*server.Server
		var serversCount int
		var searchList searchindex.SearchList[*server.Server]
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			serversCount++

			var s *server.Server
			if err := json.Unmarshal(v, &s); err != nil {
				return nil, fmt.Errorf("failed to unmarshal server: %w", err)
			}

			var optionsLen int
			if len(options.Ids) != 0 {
				optionsLen++
				if slices.Contains(options.Ids, s.Id) {
					servers = append(servers, s)
					continue
				}
			}

			if options.Enabled != nil {
				optionsLen++
				if s.Enabled == *options.Enabled {
					servers = append(servers, s)
					continue
				}
			}

			if options.CreateUserId != nil {
				optionsLen++
				if s.CreateUserId == *options.CreateUserId {
					servers = append(servers, s)
					continue
				}
			}

			if options.UpdateUserId != nil {
				optionsLen++
				if s.UpdateUserId == *options.UpdateUserId {
					servers = append(servers, s)
					continue
				}
			}

			if len(options.Query) != 0 {
				optionsLen++
				searchList = append(searchList, &searchindex.SearchItem[*server.Server]{
					Key:  s.Name,
					Data: s,
				})
				searchList = append(searchList, &searchindex.SearchItem[*server.Server]{
					Key:  s.Description,
					Data: s,
				})
			}

			if optionsLen == 0 {
				servers = append(servers, s)
			}
		}

		if len(options.Query) != 0 {
			searchIndex := searchindex.NewSearchIndex(searchList, serversCount, nil, nil, true, nil)
			matches := searchIndex.Search(searchindex.SearchParams[*server.Server]{
				Text:       options.Query,
				OutputSize: serversCount,
				Matching:   searchindex.Beginning,
			})
			servers = append(servers, matches...)
		}

		return servers, nil
	})
}

func (r *serverRepository) Create(ctx context.Context, s *server.Server) (*server.Server, error) {
	return dbTx(ctx, r.db, serverBucket, true, func(tx *bbolt.Tx, bucket *bbolt.Bucket) (*server.Server, error) {
		id := []byte(s.Id)
		if bucket.Get(id) != nil {
			return nil, server.ErrServerIdAlreadyExists
		}

		jsonState, err := json.Marshal(s)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal server: %w", err)
		}

		return s, bucket.Put(id, jsonState)
	})
}

func (r *serverRepository) Update(ctx context.Context, s *server.Server, fieldMask *server.UpdateFieldMask) (*server.Server, error) {
	return dbTx(ctx, r.db, serverBucket, false, func(tx *bbolt.Tx, bucket *bbolt.Bucket) (*server.Server, error) {
		id := []byte(s.Id)
		jsonState := bucket.Get(id)
		if jsonState == nil {
			return nil, server.ErrServerNotFound
		}

		var updatedServer *server.Server
		if err := json.Unmarshal(jsonState, &updatedServer); err != nil {
			return nil, fmt.Errorf("failed to unmarshal server: %w", err)
		}

		if fieldMask.Description {
			updatedServer.Description = s.Description
		}

		if fieldMask.Enabled {
			updatedServer.Enabled = s.Enabled
		}

		if fieldMask.Running {
			updatedServer.Running = s.Running
		}

		if fieldMask.PrivateKey {
			updatedServer.PrivateKey = s.PrivateKey
			updatedServer.PublicKey = s.PublicKey
		}

		if fieldMask.ListenPort {
			updatedServer.ListenPort = s.ListenPort
		}

		if fieldMask.FirewallMark {
			updatedServer.FirewallMark = s.FirewallMark
		}

		if fieldMask.Address {
			updatedServer.Address = s.Address
		}

		if fieldMask.DNS {
			updatedServer.DNS = s.DNS
		}

		if fieldMask.MTU {
			updatedServer.MTU = s.MTU
		}

		if fieldMask.Stats {
			updatedServer.Stats = s.Stats
		}

		if fieldMask.Hooks {
			updatedServer.Hooks = s.Hooks
		}

		if fieldMask.CreateUserId {
			updatedServer.CreateUserId = s.CreateUserId
		}

		if fieldMask.UpdateUserId {
			updatedServer.UpdateUserId = s.UpdateUserId
		}

		updatedServer.UpdatedAt = time.Now()

		jsonState, err := json.Marshal(updatedServer)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal server: %w", err)
		}

		return updatedServer, bucket.Put(id, jsonState)
	})
}

func (r *serverRepository) Delete(ctx context.Context, serverId string, deleteUserId string) (*server.Server, error) {
	return dbTx(ctx, r.db, serverBucket, false, func(tx *bbolt.Tx, bucket *bbolt.Bucket) (*server.Server, error) {
		id := []byte(serverId)
		jsonState := bucket.Get(id)
		if jsonState == nil {
			return nil, server.ErrServerNotFound
		}

		var deletedServer *server.Server
		if err := json.Unmarshal(jsonState, &deletedServer); err != nil {
			return nil, fmt.Errorf("failed to unmarshal server: %w", err)
		}

		deletedServer.DeleteUserId = deleteUserId
		deletedServer.DeletedAt = adapt.ToPointer(time.Now())

		return deletedServer, bucket.Delete(id)
	})
}
