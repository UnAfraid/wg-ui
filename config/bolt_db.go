package config

import (
	"time"
)

type BoltDB struct {
	Path    string        `required:"true"`
	Timeout time.Duration `default:"5s"`
}
