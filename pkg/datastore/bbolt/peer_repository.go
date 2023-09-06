package bbolt

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/UnAfraid/searchindex"
	"github.com/UnAfraid/wg-ui/pkg/internal/adapt"
	"github.com/UnAfraid/wg-ui/pkg/peer"
	"go.etcd.io/bbolt"
)

const (
	peerBucket = "peer"
)

type peerRepository struct {
	db *bbolt.DB
}

func NewPeerRepository(db *bbolt.DB) peer.Repository {
	return &peerRepository{
		db: db,
	}
}

func (r *peerRepository) FindOne(_ context.Context, options *peer.FindOneOptions) (*peer.Peer, error) {
	return dbView(r.db, peerBucket, false, func(tx *bbolt.Tx, bucket *bbolt.Bucket) (*peer.Peer, error) {
		if idOption := options.IdOption; idOption != nil {
			jsonState := bucket.Get([]byte(idOption.Id))
			if jsonState == nil {
				return nil, nil
			}

			var p *peer.Peer
			if err := json.Unmarshal(jsonState, &p); err != nil {
				return nil, fmt.Errorf("failed to unmarshal peer: %w", err)
			}

			return p, nil
		} else if serverIdPublicKeyOption := options.ServerIdPublicKeyOption; serverIdPublicKeyOption != nil {
			var p *peer.Peer
			c := bucket.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				if err := json.Unmarshal(v, &p); err != nil {
					return nil, fmt.Errorf("failed to unmarshal peer: %w", err)
				}
				if strings.EqualFold(p.ServerId, serverIdPublicKeyOption.ServerId) &&
					strings.EqualFold(p.PublicKey, serverIdPublicKeyOption.PublicKey) {
					return p, nil
				}
			}
		}

		return nil, nil
	})
}

func (r *peerRepository) FindAll(_ context.Context, options *peer.FindOptions) ([]*peer.Peer, error) {
	return dbView(r.db, peerBucket, false, func(tx *bbolt.Tx, bucket *bbolt.Bucket) ([]*peer.Peer, error) {
		var peers []*peer.Peer
		var peersCount int
		var searchList searchindex.SearchList[*peer.Peer]
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			peersCount++

			var p *peer.Peer
			if err := json.Unmarshal(v, &p); err != nil {
				return nil, fmt.Errorf("failed to unmarshal peer: %w", err)
			}

			var optionsLen int
			if len(options.Ids) != 0 {
				optionsLen++
				if slices.Contains(options.Ids, p.Id) {
					peers = append(peers, p)
					continue
				}
			}

			if options.ServerId != nil {
				optionsLen++
				if p.ServerId == *options.ServerId {
					peers = append(peers, p)
					continue
				}
			}

			if options.CreateUserId != nil {
				optionsLen++
				if p.CreateUserId == *options.CreateUserId {
					peers = append(peers, p)
					continue
				}
			}

			if options.UpdateUserId != nil {
				optionsLen++
				if p.UpdateUserId == *options.UpdateUserId {
					peers = append(peers, p)
					continue
				}
			}

			if len(options.Query) != 0 {
				optionsLen++
				searchList = append(searchList, &searchindex.SearchItem[*peer.Peer]{
					Key:  p.Name,
					Data: p,
				})
				searchList = append(searchList, &searchindex.SearchItem[*peer.Peer]{
					Key:  p.Description,
					Data: p,
				})
			}

			if optionsLen == 0 {
				peers = append(peers, p)
			}
		}

		if len(options.Query) != 0 {
			searchIndex := searchindex.NewSearchIndex[*peer.Peer](searchList, peersCount, nil, nil, true, nil)
			matches := searchIndex.Search(searchindex.SearchParams[*peer.Peer]{
				Text:       options.Query,
				OutputSize: peersCount,
				Matching:   searchindex.Beginning,
			})
			peers = append(peers, matches...)
		}

		return peers, nil
	})
}

func (r *peerRepository) Create(_ context.Context, p *peer.Peer) (*peer.Peer, error) {
	return dbUpdate(r.db, peerBucket, true, func(tx *bbolt.Tx, bucket *bbolt.Bucket) (*peer.Peer, error) {
		id := []byte(p.Id)
		if bucket.Get(id) != nil {
			return nil, peer.ErrPeerIdAlreadyExists
		}

		jsonState, err := json.Marshal(p)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal peer: %w", err)
		}

		return p, bucket.Put(id, jsonState)
	})
}

func (r *peerRepository) Update(_ context.Context, p *peer.Peer, fieldMask *peer.UpdateFieldMask) (*peer.Peer, error) {
	return dbUpdate(r.db, peerBucket, false, func(tx *bbolt.Tx, bucket *bbolt.Bucket) (*peer.Peer, error) {
		id := []byte(p.Id)
		jsonState := bucket.Get(id)
		if jsonState == nil {
			return nil, peer.ErrPeerNotFound
		}

		var updatedPeer *peer.Peer
		if err := json.Unmarshal(jsonState, &updatedPeer); err != nil {
			return nil, fmt.Errorf("failed to unmarshal peer: %w", err)
		}

		if fieldMask.Name {
			updatedPeer.Name = p.Name
		}

		if fieldMask.Description {
			updatedPeer.Description = p.Description
		}

		if fieldMask.PublicKey {
			updatedPeer.PublicKey = p.PublicKey
		}

		if fieldMask.Endpoint {
			updatedPeer.Endpoint = p.Endpoint
		}

		if fieldMask.AllowedIPs {
			updatedPeer.AllowedIPs = p.AllowedIPs
		}

		if fieldMask.PresharedKey {
			updatedPeer.PresharedKey = p.PresharedKey
		}

		if fieldMask.PersistentKeepalive {
			updatedPeer.PersistentKeepalive = p.PersistentKeepalive
		}

		if fieldMask.Hooks {
			updatedPeer.Hooks = p.Hooks
		}

		if fieldMask.CreateUserId {
			updatedPeer.CreateUserId = p.CreateUserId
		}

		if fieldMask.UpdateUserId {
			updatedPeer.UpdateUserId = p.UpdateUserId
		}

		updatedPeer.UpdatedAt = time.Now()

		jsonState, err := json.Marshal(updatedPeer)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal peer: %w", err)
		}

		return updatedPeer, bucket.Put(id, jsonState)
	})
}

func (r *peerRepository) Delete(_ context.Context, peerId string, deleteUserId string) (*peer.Peer, error) {
	return dbUpdate(r.db, peerBucket, false, func(tx *bbolt.Tx, bucket *bbolt.Bucket) (*peer.Peer, error) {
		id := []byte(peerId)
		jsonState := bucket.Get(id)
		if jsonState == nil {
			return nil, peer.ErrPeerNotFound
		}

		var deletedPeer *peer.Peer
		if err := json.Unmarshal(jsonState, &deletedPeer); err != nil {
			return nil, fmt.Errorf("failed to unmarshal peer: %w", err)
		}

		deletedPeer.DeleteUserId = deleteUserId
		deletedPeer.DeletedAt = adapt.ToPointer(time.Now())

		return deletedPeer, bucket.Delete(id)
	})
}
