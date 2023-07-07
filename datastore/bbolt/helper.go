package bbolt

import (
	"go.etcd.io/bbolt"
)

func dbView[T any](db *bbolt.DB, bucketName string, createBucketIfNotExists bool, callback func(*bbolt.Tx, *bbolt.Bucket) (T, error)) (result T, err error) {
	err = db.View(func(tx *bbolt.Tx) error {
		var bucket *bbolt.Bucket
		if createBucketIfNotExists {
			bucket, err = tx.CreateBucketIfNotExists([]byte(bucketName))
			if err != nil {
				return err
			}
		} else {
			bucket = tx.Bucket([]byte(bucketName))
			if bucket == nil {
				return nil
			}
		}
		result, err = callback(tx, bucket)
		return err
	})
	return result, err
}

func dbUpdate[T any](db *bbolt.DB, bucketName string, createBucketIfNotExists bool, callback func(*bbolt.Tx, *bbolt.Bucket) (T, error)) (result T, err error) {
	err = db.Update(func(tx *bbolt.Tx) error {
		var bucket *bbolt.Bucket
		if createBucketIfNotExists {
			bucket, err = tx.CreateBucketIfNotExists([]byte(bucketName))
			if err != nil {
				return err
			}
		} else {
			bucket = tx.Bucket([]byte(bucketName))
			if bucket == nil {
				return nil
			}
		}
		result, err = callback(tx, bucket)
		return err
	})
	return result, err
}
