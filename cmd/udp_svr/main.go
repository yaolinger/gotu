package main

import (
	"context"
	"flag"
	"gotu/pkg/xcommon"
	"gotu/pkg/xlog"
	"gotu/pkg/xmsg"
	"gotu/pkg/xnet"

	"go.uber.org/zap"
)

var addr = flag.String("addr", ":5000", "listen addr")

func main() {
	flag.Parse()
	ctx := context.Background()
	defer xcommon.Recover(ctx)

	svr, err := xnet.NewUDPServer(ctx, xnet.UDPSvrArgs{
		Addr:    *addr,
		Timeout: 10,
		OnConnect: func(ctx context.Context, sock xnet.Socket) interface{} {
			xlog.Get(ctx).Debug("Svr connect")
			return sock
		},
		OnDisconnect: func(ctx context.Context, state interface{}) {
			xlog.Get(ctx).Debug("Svr disconnect")
		},
		OnMsg: xmsg.ParseMsgWarp(func(ctx context.Context, arg xmsg.MsgArgs) error {
			xlog.Get(ctx).Debug("Svr recv msg", zap.String("msg", string(arg.Payload)))
			s := arg.State.(xnet.Socket)
			msg, err := xmsg.PackMsg(ctx, xmsg.PackMsgArgs{
				Payload: []byte("svr data"),
			})
			if err != nil {
				return err
			}
			if err := s.SendMsg(ctx, msg); err != nil {
				xlog.Get(ctx).Warn("Svr send msg failed.", zap.Any("err", err))
			}
			return nil
		}),
	})
	if err != nil {
		panic(err)
	}
	defer svr.Close(ctx)

	xcommon.UntilSignal(ctx)
}
