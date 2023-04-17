package bbolt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/UnAfraid/wg-ui/server"
	"github.com/twelvedata/searchindex"
	"go.etcd.io/bbolt"
	"golang.org/x/exp/slices"
)

const (
	serverBucket = "server"
)

type wgServerRepository struct {
	db *bbolt.DB
}

func NewServerRepository(db *bbolt.DB) server.Repository {
	return &wgServerRepository{
		db: db,
	}
}

func (r *wgServerRepository) FindOne(_ context.Context, options *server.FindOneOptions) (u *server.Server, err error) {
	err = r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(serverBucket))
		if bucket == nil {
			return nil
		}

		if idOption := options.IdOption; idOption != nil {
			jsonState := bucket.Get([]byte(idOption.Id))
			if jsonState == nil {
				return nil
			}

			if err := json.Unmarshal(jsonState, &u); err != nil {
				return fmt.Errorf("failed to unmarshal server: %w", err)
			}

			return nil
		} else if nameOption := options.NameOption; nameOption != nil {
			var srv *server.Server
			c := bucket.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				if err := json.Unmarshal(v, &srv); err != nil {
					return fmt.Errorf("failed to unmarshal server: %w", err)
				}
				if strings.EqualFold(srv.Name, nameOption.Name) {
					u = srv
					return nil
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *wgServerRepository) FindAll(_ context.Context, options *server.FindOptions) (servers []*server.Server, err error) {
	err = r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(serverBucket))
		if bucket == nil {
			return nil
		}

		var serversCount int
		var searchList searchindex.SearchList
		err = bucket.ForEach(func(k, v []byte) error {
			serversCount++

			var svc *server.Server
			if err := json.Unmarshal(v, &svc); err != nil {
				return fmt.Errorf("failed to unmarshal server: %w", err)
			}

			var optionsLen int
			if len(options.Ids) != 0 {
				optionsLen++
				if slices.Contains(options.Ids, svc.Id) {
					servers = append(servers, svc)
					return nil
				}
			}

			if options.Enabled != nil {
				optionsLen++
				if svc.Enabled == *options.Enabled {
					servers = append(servers, svc)
					return nil
				}
			}

			if options.CreateUserId != nil {
				optionsLen++
				if svc.CreateUserId == *options.CreateUserId {
					servers = append(servers, svc)
					return nil
				}
			}

			if options.UpdateUserId != nil {
				optionsLen++
				if svc.UpdateUserId == *options.UpdateUserId {
					servers = append(servers, svc)
					return nil
				}
			}

			if len(options.Query) != 0 {
				optionsLen++
				searchList = append(searchList, &searchindex.SearchItem{
					Key:  svc.Name,
					Data: svc,
				})
				searchList = append(searchList, &searchindex.SearchItem{
					Key:  svc.Description,
					Data: svc,
				})
			}

			if optionsLen == 0 {
				servers = append(servers, svc)
			}
			return nil
		})
		if err != nil {
			return err
		}

		if len(options.Query) != 0 {
			searchIndex := searchindex.NewSearchIndex(searchList, serversCount, nil, nil, true, nil)
			matches := searchIndex.Search(searchindex.SearchParams{
				Text:       options.Query,
				OutputSize: serversCount,
				Matching:   searchindex.Beginning,
			})
			for _, match := range matches {
				servers = append(servers, match.(*server.Server))
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return servers, nil
}

func (r *wgServerRepository) Create(_ context.Context, createServer *server.Server) (*server.Server, error) {
	err := r.db.Update(func(tx *bbolt.Tx) error {
		id := []byte(createServer.Id)
		bucket, err := tx.CreateBucketIfNotExists([]byte(serverBucket))
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}

		if bucket.Get(id) != nil {
			return server.ErrServerIdAlreadyExists
		}

		jsonState, err := json.Marshal(createServer)
		if err != nil {
			return fmt.Errorf("failed to marshal server: %w", err)
		}

		return bucket.Put(id, jsonState)
	})
	if err != nil {
		return nil, err
	}
	return createServer, nil
}

func (r *wgServerRepository) Update(_ context.Context, updateServer *server.Server, fieldMask *server.UpdateFieldMask) (updatedServer *server.Server, err error) {
	err = r.db.Update(func(tx *bbolt.Tx) error {
		id := []byte(updateServer.Id)
		bucket, err := tx.CreateBucketIfNotExists([]byte(serverBucket))
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}

		jsonState := bucket.Get(id)
		if jsonState == nil {
			return server.ErrServerNotFound
		}

		if err := json.Unmarshal(jsonState, &updatedServer); err != nil {
			return fmt.Errorf("failed to unmarshal server: %w", err)
		}

		if fieldMask.Description {
			updatedServer.Description = updateServer.Description
		}

		if fieldMask.Enabled {
			updatedServer.Enabled = updateServer.Enabled
		}

		if fieldMask.Running {
			updatedServer.Running = updateServer.Running
		}

		if fieldMask.PublicKey {
			updatedServer.PublicKey = updateServer.PublicKey
		}

		if fieldMask.ListenPort {
			updatedServer.ListenPort = updateServer.ListenPort
		}

		if fieldMask.FirewallMark {
			updatedServer.FirewallMark = updateServer.FirewallMark
		}

		if fieldMask.Address {
			updatedServer.Address = updateServer.Address
		}

		if fieldMask.DNS {
			updatedServer.DNS = updateServer.DNS
		}

		if fieldMask.MTU {
			updatedServer.MTU = updateServer.MTU
		}

		if fieldMask.Hooks {
			updatedServer.Hooks = updateServer.Hooks
		}

		if fieldMask.UpdateUserId {
			updatedServer.UpdateUserId = updateServer.UpdateUserId
		}

		updatedServer.UpdatedAt = time.Now()

		jsonState, err = json.Marshal(updatedServer)
		if err != nil {
			return fmt.Errorf("failed to unmarshal server: %w", err)
		}

		return bucket.Put(id, jsonState)
	})
	if err != nil {
		return nil, err
	}
	return updatedServer, nil
}

func (r *wgServerRepository) Delete(_ context.Context, serverId string, deleteUserId string) (deletedServer *server.Server, err error) {
	err = r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(serverBucket))
		if bucket == nil {
			return nil
		}

		id := []byte(serverId)
		jsonState := bucket.Get(id)
		if jsonState == nil {
			return server.ErrServerNotFound
		}

		if err := json.Unmarshal(jsonState, &deletedServer); err != nil {
			return fmt.Errorf("failed to unmarshal server: %w", err)
		}

		deletedServer.DeleteUserId = deleteUserId
		deletedServer.DeletedAt = adapt.ToPointer(time.Now())

		return bucket.Delete(id)
	})
	if err != nil {
		return nil, err
	}
	return deletedServer, nil
}
