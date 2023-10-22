package bbolt

import (
	"context"

	"go.etcd.io/bbolt"

	"github.com/UnAfraid/wg-ui/pkg/dbx"
)

func dbTx[T any](ctx context.Context, db *bbolt.DB, bucketName string, createBucketIfNotExists bool, callback func(*bbolt.Tx, *bbolt.Bucket) (T, error)) (T, error) {
	return dbx.InBBoltTransactionScopeWithResult(ctx, db, func(ctx context.Context, tx *bbolt.Tx) (result T, err error) {
		var bucket *bbolt.Bucket
		if createBucketIfNotExists {
			bucket, err = tx.CreateBucketIfNotExists([]byte(bucketName))
			if err != nil {
				return result, err
			}
		} else {
			bucket = tx.Bucket([]byte(bucketName))
			if bucket == nil {
				return result, nil
			}
		}
		return callback(tx, bucket)
	})
}
