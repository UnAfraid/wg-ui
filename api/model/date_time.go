package model

import (
	"errors"
	"io"
	"strconv"
	"time"

	"github.com/99designs/gqlgen/graphql"
)

const dateTimeLayout = time.RFC3339

func MarshalDateTime(t time.Time) graphql.Marshaler {
	if t.IsZero() {
		return graphql.Null
	}
	return graphql.WriterFunc(func(w io.Writer) {
		if _, err := io.WriteString(w, strconv.Quote(t.Format(dateTimeLayout))); err != nil {
			panic(err)
		}
	})
}

func UnmarshalDateTime(v interface{}) (time.Time, error) {
	if fullDateTime, ok := v.(string); ok {
		return time.Parse(dateTimeLayout, fullDateTime)
	}
	return time.Time{}, errors.New("date should be a date-time RFC3339 formatted string")
}
