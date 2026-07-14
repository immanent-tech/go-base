// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package client

import (
	"fmt"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/joshuar/go-base/config"
)

const (
	configEnvPrefix = "CLIENT_"
)

type Config struct {
	// UserAgent is the string which the `User-Agent` request header will be set to for client requests.
	UserAgent string `koanf:"user_agent" validate:"required"`
	// DefaultHTTPRequestTimeout is the maximum time allowed for a background HTTP request to execute.
	DefaultHTTPRequestTimeout config.Duration `koanf:"request_timeout" validation:"omitempty,validateFn"`
	// DefaultRequestRetries is the default number of retries for API requests.
	DefaultRequestRetries int `koanf:"request_retries" validation:"omitempty,gt=0"`
}

var cfg *Config

var client *resty.Client

var Load = sync.OnceValues(func() (*resty.Client, error) {
	cfg = &Config{
		UserAgent:             config.GetAppName() + "/" + config.GetVersion(),
		DefaultRequestRetries: 3,
	}

	if err := cfg.DefaultHTTPRequestTimeout.UnmarshalText([]byte("45s")); err != nil {
		return nil, fmt.Errorf("set default timeout: %w", err)
	}

	if err := config.Load(configEnvPrefix, cfg); err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	client = resty.New().
		SetHeader("User-Agent", cfg.UserAgent).
		SetHeader("Accept", "*/*").
		SetHeader("Accept-Encoding", "gzip, deflate")
	return client, nil
})
