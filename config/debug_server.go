package config

import (
	"fmt"
)

type DebugServer struct {
	Enabled bool   `default:"false"`
	Host    string `default:"127.0.0.1"`
	Port    uint16 `default:"6060"`
}

func (s *DebugServer) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}
