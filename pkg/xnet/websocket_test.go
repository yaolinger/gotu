package xnet_test

import (
	"context"
	"fmt"
	"gonet/pkg/xlog"
	"gonet/pkg/xmsg"
	"gonet/pkg/xnet"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestWebsocket(t *testing.T) {
	ctx := context.Background()
	addr := ":9999"
	path := "/"
	var wg sync.WaitGroup
	svr := xnet.NewWSServer(ctx, xnet.WSSvrArgs{Addr: addr, Path: path,
		OnConnect: func(ctx context.Context, sock xnet.Socket) interface{} { return sock },
		OnDisconnect: func(ctx context.Context, state interface{}) {
			xlog.Get(ctx).Info("Cli disconnect")
		},
		OnMsg: xmsg.ParseMsgWarp(func(ctx context.Context, arg xmsg.MsgArgs) error {
			xlog.Get(ctx).Info("Svr recv msg", zap.Any("msg", string(arg.Payload)))
			sock := arg.State.(xnet.Socket)
			msg, err := xmsg.PackMsg(ctx, xmsg.PackMsgArgs{
				Payload: []byte("svr data"),
			})
			if err != nil {
				return err
			}
			return sock.SendMsg(ctx, msg)

		})})

	time.Sleep(1 * time.Second)

	cli, err := xnet.NewWSClient(ctx, xnet.WSCliArgs{
		Addr:      addr,
		Path:      path,
		OnConnect: func(ctx context.Context, fs xnet.Socket) interface{} { return nil },
		OnDisconnect: func(ctx context.Context, state interface{}) {
			xlog.Get(ctx).Info("Svr disconnect")
		},
		OnMsg: xmsg.ParseMsgWarp(func(ctx context.Context, arg xmsg.MsgArgs) error {
			defer wg.Done()
			xlog.Get(ctx).Info("Cli recv msg", zap.Any("msg", string(arg.Payload)))
			return nil
		}),
	})

	if err != nil {
		panic(err)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
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

	wg.Wait()

	cli.Close(ctx)

	time.Sleep(1 * time.Second)

	svr.Close(ctx)

}
