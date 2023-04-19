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

type TcpState struct {
	sock xnet.Socket
}

func main() {
	flag.Parse()
	ctx := context.Background()
	defer xcommon.Recover(ctx)

	svr, err := xnet.NewTCPServer(ctx, xnet.TCPSvrArgs{
		Addr: *addr,
		OnConnect: func(ctx context.Context, sock xnet.Socket) interface{} {
			//xlog.Get(ctx).Info("Tcp socket connect.", zap.Any("remote", sock.RemoteAddr()), zap.Any("local", sock.LocalAddr()))
			return &TcpState{sock: sock}
		},
		OnDisconnect: func(ctx context.Context, state interface{}) {
			//s := state.(*TcpState)
			//xlog.Get(ctx).Info("Tcp socket disconnect", zap.Any("remote", s.sock.RemoteAddr()), zap.Any("local", s.sock.LocalAddr()))
		},
		OnMsg: xmsg.ParseMsgWarp(func(ctx context.Context, arg xmsg.MsgArgs) error {
			s := arg.State.(*TcpState)
			msg, err := xmsg.PackMsg(ctx, xmsg.PackMsgArgs{
				Payload: arg.Payload,
			})
			if err != nil {
				return err
			}
			if err := s.sock.SendMsg(ctx, msg); err != nil {
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
