package model

import (
	"context"
	"errors"

	"github.com/99designs/gqlgen/graphql"
	"github.com/sirupsen/logrus"
)

func resolverHasArgumentField(ctx context.Context, fieldPath ...string) bool {
	oc := graphql.GetOperationContext(ctx)
	fc := graphql.GetFieldContext(ctx)
	arguments := fc.Field.ArgumentMap(oc.Variables)
	hasField, err := interfaceHasField(arguments, fieldPath...)
	if err != nil {
		logrus.
			WithError(err).
			WithField("arguments", arguments).
			WithField("fieldPath", fieldPath).
			Error("failed to check if resolver has argument field")
		return false
	}
	return hasField
}

func interfaceHasField(v interface{}, fieldPath ...string) (bool, error) {
	for i, field := range fieldPath {
		switch vt := v.(type) {
		case nil:
			return false, nil
		case map[string]interface{}:
			var ok bool
			v, ok = vt[field]
			if !ok {
				return false, nil
			}
		case []interface{}:
			for _, av := range vt {
				hasField, err := interfaceHasField(av, fieldPath[i:]...)
				if err != nil {
					return false, err
				}

				if hasField {
					return true, nil
				}
			}
			return false, nil
		default:
			return false, errors.New("invalid field tried to go to non map or array type")
		}
	}
	return true, nil
}
