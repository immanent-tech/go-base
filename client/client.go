// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package client

import (
	"fmt"
	"log/slog"

	"github.com/go-resty/resty/v2"
	"github.com/immanent-tech/go-base/config"
)

func New() *resty.Client {
	userAgent := "Go-Base/Unknown"
	if baseCfg, err := config.LoadAppConfig(); err == nil {
		userAgent = baseCfg.AppName + "/" + baseCfg.Version
	}
	return resty.New().
		SetHeader("User-Agent", userAgent).
		SetHeader("Accept", "*/*").
		SetHeader("Accept-Encoding", "gzip, deflate").
		SetRedirectPolicy(resty.FlexibleRedirectPolicy(3)).
		SetLogger(&logger{Logger: slog.Default()})
}

type logger struct {
	*slog.Logger
}

func (l *logger) Errorf(format string, v ...any) {
	l.Error(fmt.Sprintf(format, v...))
}

func (l *logger) Warnf(format string, v ...any) {
	l.Warn(fmt.Sprintf(format, v...))
}

func (l *logger) Debugf(format string, v ...any) {
	l.Debug(fmt.Sprintf(format, v...))
}
