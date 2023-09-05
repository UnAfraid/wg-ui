package model

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

var ErrIDNotString = errors.New("ID is not a string")

type ID struct {
	Kind  IdKind
	Value string
}

func StringID(idKind IdKind, id string) ID {
	return ID{
		Kind:  idKind,
		Value: id,
	}
}

func (id ID) Equal(other ID) bool {
	return id.Kind == other.Kind && id.Value == other.Value
}

func (id *ID) String(idKind IdKind) (string, error) {
	if id == nil {
		return "", nil
	}

	if err := id.Validate(idKind); err != nil {
		return "", err
	}

	return id.Value, nil
}

func (id *ID) Validate(idKind IdKind) error {
	if id.Kind != idKind {
		return fmt.Errorf("%s ID needed but %s is %s", idKind, id.Base64(), id.Kind)
	}
	return nil
}

func (id *ID) Base64() string {
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", id.Kind, id.Value)))
}

func (id ID) MarshalGQL(w io.Writer) {
	if _, err := io.WriteString(w, strconv.Quote(id.Base64())); err != nil {
		panic(err)
	}
}

func (id *ID) UnmarshalGQL(v interface{}) error {
	base64Id, ok := v.(string)
	if !ok {
		return ErrIDNotString
	}

	bytes, err := base64.StdEncoding.DecodeString(base64Id)
	if err != nil {
		return fmt.Errorf("failed to decode id '%s': %w", base64Id, err)
	}

	idSplit := strings.Split(string(bytes), ":")
	if len(idSplit) != 2 {
		return fmt.Errorf("can not decode invalid id '%s'", base64Id)
	}

	id.Kind = IdKind(idSplit[0])
	id.Value = idSplit[1]

	return nil
}
