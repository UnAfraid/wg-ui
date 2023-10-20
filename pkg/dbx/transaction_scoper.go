package dbx

import (
	"context"
)

type TransactionScoper interface {
	InTransactionScope(ctx context.Context, transactionScope func(ctx context.Context) error) error
}

func InTransactionScopeWithResult[T any](ctx context.Context, transactionScoper TransactionScoper, transactionScope func(ctx context.Context) (T, error)) (result T, err error) {
	err = transactionScoper.InTransactionScope(ctx, func(ctx context.Context) error {
		result, err = transactionScope(ctx)
		return err
	})
	return result, err
}
