package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	BoltDB                     *BoltDB       `split_words:"true"`
	HttpServer                 *HttpServer   `split_words:"true"`
	DebugServer                *DebugServer  `split_words:"true"`
	Initial                    *Initial      `required:"true"`
	CorsAllowedOrigins         []string      `split_words:"true" default:"*"`
	CorsAllowCredentials       bool          `split_words:"true" default:"true"`
	CorsAllowPrivateNetwork    bool          `split_words:"true" default:"false"`
	SubscriptionAllowedOrigins []string      `split_words:"true" default:"*"`
	JwtSecret                  string        `required:"true" split_words:"true"`
	JwtDuration                time.Duration `split_words:"true" default:"8h"`
}

func Load(prefix string) (*Config, error) {
	prefix = strings.ToUpper(prefix)
	prefix = strings.ReplaceAll(prefix, "-", "_")
	prefix = strings.ReplaceAll(prefix, " ", "_")
	var config Config
	if err := envconfig.Process(prefix, &config); err != nil {
		return nil, fmt.Errorf("failed to process env config: %w", err)
	}
	return &config, nil
}
