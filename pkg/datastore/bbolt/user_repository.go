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

	"github.com/UnAfraid/wg-ui/pkg/user"
)

const (
	userBucket = "user"
)

type userRepository struct {
	db *bbolt.DB
}

func NewUserRepository(db *bbolt.DB) user.Repository {
	return &userRepository{
		db: db,
	}
}

func (r *userRepository) FindOne(_ context.Context, options *user.FindOneOptions) (*user.User, error) {
	return dbView(r.db, userBucket, false, func(tx *bbolt.Tx, bucket *bbolt.Bucket) (*user.User, error) {
		if idOption := options.IdOption; idOption != nil {
			jsonState := bucket.Get([]byte(idOption.Id))
			if jsonState == nil {
				return nil, nil
			}

			var u *user.User
			if err := json.Unmarshal(jsonState, &u); err != nil {
				return nil, fmt.Errorf("failed to unmarshal user: %w", err)
			}

			if u.DeletedAt != nil && !idOption.WithDeleted {
				u = nil
			}

			return u, nil
		} else if emailOption := options.EmailOption; emailOption != nil {
			var u *user.User
			c := bucket.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				if err := json.Unmarshal(v, &u); err != nil {
					return nil, fmt.Errorf("failed to unmarshal user: %w", err)
				}
				if strings.EqualFold(u.Email, emailOption.Email) {
					return u, nil
				}
			}
		}

		return nil, nil
	})
}

func (r *userRepository) FindAll(_ context.Context, options *user.FindOptions) ([]*user.User, error) {
	return dbView(r.db, userBucket, false, func(tx *bbolt.Tx, bucket *bbolt.Bucket) ([]*user.User, error) {
		var users []*user.User
		var usersCount int
		var searchList searchindex.SearchList[*user.User]
		c := bucket.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			usersCount++
			var u *user.User
			if err := json.Unmarshal(v, &u); err != nil {
				return nil, fmt.Errorf("failed to unmarshal user: %w", err)
			}

			var optionsLen int
			if len(options.Ids) != 0 {
				optionsLen++
				if slices.Contains(options.Ids, u.Id) {
					users = append(users, u)
					continue
				}
			}

			if len(options.Query) != 0 {
				optionsLen++
				searchList = append(searchList, &searchindex.SearchItem[*user.User]{
					Key:  u.Email,
					Data: u,
				})
			}

			if optionsLen == 0 {
				users = append(users, u)
			}
		}

		if len(options.Query) != 0 {
			searchIndex := searchindex.NewSearchIndex(searchList, usersCount, nil, nil, true, nil)
			matches := searchIndex.Search(searchindex.SearchParams[*user.User]{
				Text:       options.Query,
				OutputSize: usersCount,
				Matching:   searchindex.Beginning,
			})
			users = append(users, matches...)
		}

		return users, nil
	})
}

func (r *userRepository) Create(_ context.Context, u *user.User) (*user.User, error) {
	return dbUpdate(r.db, userBucket, true, func(tx *bbolt.Tx, bucket *bbolt.Bucket) (*user.User, error) {
		id := []byte(u.Id)
		if bucket.Get(id) != nil {
			return nil, user.ErrUserIdAlreadyExists
		}

		jsonState, err := json.Marshal(u)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal user: %w", err)
		}

		return u, bucket.Put(id, jsonState)
	})
}

func (r *userRepository) Update(_ context.Context, u *user.User, fieldMask *user.UpdateFieldMask) (*user.User, error) {
	return dbUpdate(r.db, userBucket, false, func(tx *bbolt.Tx, bucket *bbolt.Bucket) (*user.User, error) {
		id := []byte(u.Id)
		jsonState := bucket.Get(id)
		if jsonState == nil {
			return nil, user.ErrUserNotFound
		}

		var updatedUser *user.User
		if err := json.Unmarshal(jsonState, &updatedUser); err != nil {
			return nil, fmt.Errorf("failed to unmarshal user: %w", err)
		}

		if fieldMask.Email {
			updatedUser.Email = u.Email
		}

		if fieldMask.Password {
			updatedUser.Password = u.Password
		}

		updatedUser.UpdatedAt = time.Now()

		jsonState, err := json.Marshal(updatedUser)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal user: %w", err)
		}

		return updatedUser, bucket.Put(id, jsonState)
	})
}

func (r *userRepository) Delete(_ context.Context, userId string) (*user.User, error) {
	return dbUpdate(r.db, userBucket, false, func(tx *bbolt.Tx, bucket *bbolt.Bucket) (*user.User, error) {
		id := []byte(userId)
		jsonState := bucket.Get(id)
		if jsonState == nil {
			return nil, user.ErrUserNotFound
		}

		var deletedUser *user.User
		if err := json.Unmarshal(jsonState, &deletedUser); err != nil {
			return nil, fmt.Errorf("failed to unmarshal user: %w", err)
		}

		now := time.Now()
		deletedUser.DeletedAt = &now

		return deletedUser, bucket.Delete(id)
	})
}
