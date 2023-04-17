package bbolt

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/UnAfraid/wg-ui/user"
	"github.com/twelvedata/searchindex"
	"go.etcd.io/bbolt"
	"golang.org/x/exp/slices"
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

func (r *userRepository) FindOne(_ context.Context, options *user.FindOneOptions) (u *user.User, err error) {
	err = r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(userBucket))
		if bucket == nil {
			return nil
		}

		if idOption := options.IdOption; idOption != nil {
			jsonState := bucket.Get([]byte(idOption.Id))
			if jsonState == nil {
				return nil
			}

			if err := json.Unmarshal(jsonState, &u); err != nil {
				return fmt.Errorf("failed to unmarshal user: %w", err)
			}

			if u.DeletedAt != nil && !idOption.WithDeleted {
				u = nil
			}

			return nil
		} else if emailOption := options.EmailOption; emailOption != nil {
			var usr *user.User
			c := bucket.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				if err := json.Unmarshal(v, &usr); err != nil {
					return fmt.Errorf("failed to unmarshal user: %w", err)
				}
				if strings.EqualFold(usr.Email, emailOption.Email) {
					u = usr
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

func (r *userRepository) FindAll(_ context.Context, options *user.FindOptions) (users []*user.User, err error) {
	err = r.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(userBucket))
		if bucket == nil {
			return nil
		}

		var usersCount int
		var searchList searchindex.SearchList
		err = bucket.ForEach(func(k, v []byte) error {
			usersCount++
			var usr *user.User
			if err := json.Unmarshal(v, &usr); err != nil {
				return fmt.Errorf("failed to unmarshal user: %w", err)
			}

			var optionsLen int
			if len(options.Ids) != 0 {
				optionsLen++
				if slices.Contains(options.Ids, usr.Id) {
					users = append(users, usr)
					return nil
				}
			}

			if len(options.Query) != 0 {
				optionsLen++
				searchList = append(searchList, &searchindex.SearchItem{
					Key:  usr.Email,
					Data: usr,
				})
			}

			if optionsLen == 0 {
				users = append(users, usr)
			}
			return nil
		})
		if err != nil {
			return err
		}

		if len(options.Query) != 0 {
			searchIndex := searchindex.NewSearchIndex(searchList, usersCount, nil, nil, true, nil)
			matches := searchIndex.Search(searchindex.SearchParams{
				Text:       options.Query,
				OutputSize: usersCount,
				Matching:   searchindex.Beginning,
			})
			for _, match := range matches {
				users = append(users, match.(*user.User))
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return users, nil
}

func (r *userRepository) Create(_ context.Context, usr *user.User) (*user.User, error) {
	err := r.db.Update(func(tx *bbolt.Tx) error {
		id := []byte(usr.Id)
		bucket, err := tx.CreateBucketIfNotExists([]byte(userBucket))
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}

		if bucket.Get(id) != nil {
			return user.ErrUserIdAlreadyExists
		}

		jsonState, err := json.Marshal(usr)
		if err != nil {
			return fmt.Errorf("failed to marshal user: %w", err)
		}

		return bucket.Put(id, jsonState)
	})
	if err != nil {
		return nil, err
	}
	return usr, nil
}

func (r *userRepository) Update(_ context.Context, updateUser *user.User, fieldMask *user.UpdateFieldMask) (updatedUser *user.User, err error) {
	err = r.db.Update(func(tx *bbolt.Tx) error {
		id := []byte(updateUser.Id)
		bucket, err := tx.CreateBucketIfNotExists([]byte(userBucket))
		if err != nil {
			return fmt.Errorf("failed to create bucket: %w", err)
		}

		jsonState := bucket.Get(id)
		if jsonState == nil {
			return user.ErrUserNotFound
		}

		if err := json.Unmarshal(jsonState, &updatedUser); err != nil {
			return fmt.Errorf("failed to unmarshal user: %w", err)
		}

		if fieldMask.Email {
			updatedUser.Email = updateUser.Email
		}

		if fieldMask.Password {
			updatedUser.Password = updateUser.Password
		}

		updatedUser.UpdatedAt = time.Now()

		jsonState, err = json.Marshal(updatedUser)
		if err != nil {
			return fmt.Errorf("failed to marshal user: %w", err)
		}

		return bucket.Put(id, jsonState)
	})
	if err != nil {
		return nil, err
	}
	return updatedUser, nil
}

func (r *userRepository) Delete(_ context.Context, userId string) (deletedUser *user.User, err error) {
	err = r.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte(userBucket))
		if bucket == nil {
			return nil
		}

		id := []byte(userId)
		jsonState := bucket.Get(id)
		if jsonState == nil {
			return user.ErrUserNotFound
		}

		if err := json.Unmarshal(jsonState, &deletedUser); err != nil {
			return fmt.Errorf("failed to unmarshal user: %w", err)
		}

		now := time.Now()
		deletedUser.DeletedAt = &now

		return bucket.Delete(id)
	})
	if err != nil {
		return nil, err
	}
	return deletedUser, nil
}
