package dbx

import (
	"context"
	"errors"

	"go.etcd.io/bbolt"
)

type contextKey struct{ name string }

var bboltTxKey = contextKey{name: "bboltTxKey"}

type bboltTransactionScoper struct {
	db *bbolt.DB
}

func NewBBoltTransactionScoper(db *bbolt.DB) TransactionScoper {
	return &bboltTransactionScoper{
		db: db,
	}
}

func (bts *bboltTransactionScoper) InTransactionScope(ctx context.Context, transactionScope func(ctx context.Context) error) (err error) {
	return InBBoltTransactionScope(ctx, bts.db, func(ctx context.Context, tx *bbolt.Tx) error {
		return transactionScope(ctx)
	})
}

func InBBoltTransactionScope(ctx context.Context, db *bbolt.DB, transactionScope func(ctx context.Context, tx *bbolt.Tx) error) (retErr error) {
	tx, transactionCloser, err := useOrStartBBoltTransaction(ctx, db)
	if err != nil {
		return err
	}

	defer func() {
		retErr = transactionCloser(retErr)
	}()

	return transactionScope(context.WithValue(ctx, bboltTxKey, tx), tx)
}

func InBBoltTransactionScopeWithResult[T any](ctx context.Context, db *bbolt.DB, transactionScope func(ctx context.Context, tx *bbolt.Tx) (T, error)) (result T, err error) {
	tx, transactionCloser, err := useOrStartBBoltTransaction(ctx, db)
	if err != nil {
		return result, err
	}

	defer func() {
		err = transactionCloser(err)
	}()

	return transactionScope(context.WithValue(ctx, bboltTxKey, tx), tx)
}

func useOrStartBBoltTransaction(ctx context.Context, db *bbolt.DB) (*bbolt.Tx, func(err error) error, error) {
	tx, ok := ctx.Value(bboltTxKey).(*bbolt.Tx)
	if !ok {
		tx, err := db.Begin(true)
		if err != nil {
			return nil, nil, err
		}

		transactionScope := func(err error) error {
			if err != nil {
				if txErr := tx.Rollback(); txErr != nil {
					err = errors.Join(err, txErr)
				}
			} else {
				if txErr := tx.Commit(); txErr != nil {
					err = txErr
				}
			}
			return err
		}

		return tx, transactionScope, nil
	} else {
		transactionCloser := func(err error) error {
			return err
		}
		return tx, transactionCloser, nil
	}
}
