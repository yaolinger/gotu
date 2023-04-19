package main

import (
	"context"
	"gotu/pkg/xcommon"
	"gotu/pkg/xenv"
	"gotu/pkg/xlog"

	"go.uber.org/zap"
)

type Config struct {
	Addr string `env:"ADDR" envDefault:":5000"`
	Num  int    `env:"NUM" envDefault:"1"`
}

func main() {
	ctx := context.Background()
	defer xcommon.Recover(ctx)

	cfg := &Config{}
	if err := xenv.EnvLoad(cfg); err != nil {
		panic(err)
	}

	xlog.Get(ctx).Info("Load config", zap.Any("config", cfg))

	cli := NewTClient(ctx, ClientArgs{Num: cfg.Num, Addr: cfg.Addr})

	cli.Start(ctx)
}
