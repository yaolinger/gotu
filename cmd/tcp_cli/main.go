package main

import (
	"context"
	"flag"
	"gotu/pkg/xcommon"
)

var addr = flag.String("addr", "172.17.0.2:5000", "connect addr")
var num = flag.Int("num", 30000, "tcp cli num")

func main() {
	flag.Parse()

	ctx := context.Background()
	defer xcommon.Recover(ctx)

	cli := NewTClient(ctx, ClientArgs{Num: *num, Addr: *addr})

	cli.Start(ctx)
}
