package config

import (
	"fmt"
)

type HttpServer struct {
	Host                    string `default:""`
	Port                    uint16 `default:"8080"`
	APQCacheEnabled         bool   `default:"false" split_words:"true"`
	TracingEnabled          bool   `default:"false" split_words:"true"`
	FrontendEnabled         bool   `default:"true" split_words:"true"`
	GraphiQLEnabled         bool   `default:"true" split_words:"true"`
	GraphiQLEndpoint        string `default:"/graphiql" split_words:"true"`
	GraphiQLVersion         string `default:"default" split_words:"true"`
	SandboxExplorerEnabled  bool   `default:"true" split_words:"true"`
	SandboxExplorerEndpoint string `default:"/sandbox" split_words:"true"`
	PlaygroundEnabled       bool   `default:"false" split_words:"true"`
	PlaygroundEndpoint      string `default:"/playground" split_words:"true"`
	AltairEnabled           bool   `default:"false" split_words:"true"`
	AltairEndpoint          string `default:"/altair" split_words:"true"`
	VoyagerEnabled          bool   `default:"false" split_words:"true"`
	VoyagerEndpoint         string `default:"/voyager" split_words:"true"`
}

func (s *HttpServer) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}
