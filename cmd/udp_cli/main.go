package main

import (
	"context"
	"flag"
	"fmt"
	"gotu/pkg/xcommon"
	"gotu/pkg/xlog"
	"gotu/pkg/xmsg"
	"gotu/pkg/xnet"
	"time"

	"go.uber.org/zap"
)

var addr = flag.String("addr", ":6000", "connect addr")

func main() {
	flag.Parse()

	ctx := context.Background()
	defer xcommon.Recover(ctx)

	cli, err := xnet.NewUDPClient(ctx, xnet.UDPCliArgs{
		Addr:    *addr,
		Timeout: 10,
		OnConnect: func(ctx context.Context, sock xnet.Socket) interface{} {
			xlog.Get(ctx).Debug("Cli connect")
			return nil
		},
		OnDisconnect: func(ctx context.Context, state interface{}) {
			xlog.Get(ctx).Debug("Cli disconnect")
		},
		OnMsg: xmsg.ParseMsgWarp(func(ctx context.Context, arg xmsg.MsgArgs) error {
			xlog.Get(ctx).Debug("Cli recv msg", zap.String("msg", string(arg.Payload)))
			return nil
		}),
	})
	if err != nil {
		panic(err)
	}

	for i := 0; i < 10000; i++ {
		msg, err := xmsg.PackMsg(ctx, xmsg.PackMsgArgs{
			Payload: []byte(fmt.Sprintf("cli data %v", i)),
		})
		if err != nil {
			panic(err)
		}
		if err := cli.SendMsg(ctx, msg); err != nil {
			xlog.Get(ctx).Warn("Cli send msg failed.", zap.Any("err", err))
		}
		time.Sleep(100 * time.Millisecond)
	}
	cli.Close(ctx)
}
