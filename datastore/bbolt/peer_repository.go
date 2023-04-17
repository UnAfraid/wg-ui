package bbolt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/UnAfraid/wg-ui/internal/adapt"
	"github.com/UnAfraid/wg-ui/peer"
	"github.com/twelvedata/searchindex"
	"go.etcd.io/bbolt"
	"golang.org/x/exp/slices"
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

func (r *peerRepository) FindOne(_ context.Context, options *peer.FindOneOptions) (foundPeer *peer.Peer, err error) {
	err = r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(peerBucket))
		if bucket == nil {
			return nil
		}

		if idOption := options.IdOption; idOption != nil {
			jsonState := bucket.Get([]byte(idOption.Id))
			if jsonState == nil {
				return nil
			}

			if err := json.Unmarshal(jsonState, &foundPeer); err != nil {
				return fmt.Errorf("failed to unmarshal peer: %w", err)
			}

			return nil
		} else if serverIdPublicKeyOption := options.ServerIdPublicKeyOption; serverIdPublicKeyOption != nil {
			var currentPeer *peer.Peer
			c := bucket.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				if err := json.Unmarshal(v, &currentPeer); err != nil {
					return fmt.Errorf("failed to unmarshal peer: %w", err)
				}
				if strings.EqualFold(currentPeer.ServerId, serverIdPublicKeyOption.ServerId) && strings.EqualFold(currentPeer.PublicKey, serverIdPublicKeyOption.PublicKey) {
					foundPeer = currentPeer
					return nil
				}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return foundPeer, nil
}

func (r *peerRepository) FindAll(_ context.Context, options *peer.FindOptions) (peers []*peer.Peer, err error) {
	err = r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(peerBucket))
		if bucket == nil {
			return nil
		}

		var peersCount int
		var searchList searchindex.SearchList
		err = bucket.ForEach(func(k, v []byte) error {
			peersCount++

			var p *peer.Peer
			if err := json.Unmarshal(v, &p); err != nil {
				return fmt.Errorf("failed to unmarshal peer: %w", err)
			}

			var optionsLen int
			if len(options.Ids) != 0 {
				optionsLen++
				if slices.Contains(options.Ids, p.Id) {
					peers = append(peers, p)
					return nil
				}
			}

			if options.ServerId != nil {
				optionsLen++
				if p.ServerId == *options.ServerId {
					peers = append(peers, p)
					return nil
				}
			}

			if options.CreateUserId != nil {
				optionsLen++
				if p.CreateUserId == *options.CreateUserId {
					peers = append(peers, p)
					return nil
				}
			}

			if options.UpdateUserId != nil {
				optionsLen++
				if p.UpdateUserId == *options.UpdateUserId {
					peers = append(peers, p)
					return nil
				}
			}

			if len(options.Query) != 0 {
				optionsLen++
				searchList = append(searchList, &searchindex.SearchItem{
					Key:  p.Name,
					Data: p,
				})
				searchList = append(searchList, &searchindex.SearchItem{
					Key:  p.Description,
					Data: p,
				})
			}

			if optionsLen == 0 {
				peers = append(peers, p)
			}
			return nil
		})
		if err != nil {
			return err
		}

		if len(options.Query) != 0 {
			searchIndex := searchindex.NewSearchIndex(searchList, peersCount, nil, nil, true, nil)
			matches := searchIndex.Search(searchindex.SearchParams{
				Text:       options.Query,
				OutputSize: peersCount,
				Matching:   searchindex.Beginning,
			})
			for _, match := range matches {
				peers = append(peers, match.(*peer.Peer))
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return peers, nil
}

func (r *peerRepository) Create(_ context.Context, p *peer.Peer) (*peer.Peer, error) {
	err := r.db.Update(func(tx *bbolt.Tx) error {
		id := []byte(p.Id)
		bucket, err := tx.CreateBucketIfNotExists([]byte(peerBucket))
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}

		if bucket.Get(id) != nil {
			return peer.ErrPeerIdAlreadyExists
		}

		jsonState, err := json.Marshal(p)
		if err != nil {
			return fmt.Errorf("failed to marshal peer: %w", err)
		}

		return bucket.Put(id, jsonState)
	})
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (r *peerRepository) Update(_ context.Context, p *peer.Peer, fieldMask *peer.UpdateFieldMask) (updatedPeer *peer.Peer, err error) {
	err = r.db.Update(func(tx *bbolt.Tx) error {
		id := []byte(p.Id)
		bucket, err := tx.CreateBucketIfNotExists([]byte(peerBucket))
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}

		jsonState := bucket.Get(id)
		if jsonState == nil {
			return peer.ErrPeerNotFound
		}

		if err := json.Unmarshal(jsonState, &updatedPeer); err != nil {
			return fmt.Errorf("failed to unmarshal peer: %w", err)
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

		if fieldMask.UpdateUserId {
			updatedPeer.UpdateUserId = p.UpdateUserId
		}

		updatedPeer.UpdatedAt = time.Now()

		jsonState, err = json.Marshal(updatedPeer)
		if err != nil {
			return fmt.Errorf("failed to marshal peer: %w", err)
		}

		return bucket.Put(id, jsonState)
	})
	if err != nil {
		return nil, err
	}
	return updatedPeer, nil
}

func (r *peerRepository) Delete(_ context.Context, peerId string, deleteUserId string) (deletedPeer *peer.Peer, err error) {
	err = r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(peerBucket))
		if bucket == nil {
			return nil
		}

		id := []byte(peerId)
		jsonState := bucket.Get(id)
		if jsonState == nil {
			return peer.ErrPeerNotFound
		}

		if err := json.Unmarshal(jsonState, &deletedPeer); err != nil {
			return fmt.Errorf("failed to unmarshal peer: %w", err)
		}

		deletedPeer.DeleteUserId = deleteUserId
		deletedPeer.DeletedAt = adapt.ToPointer(time.Now())

		return bucket.Delete(id)
	})
	if err != nil {
		return nil, err
	}
	return deletedPeer, nil
}
