package xnet_test

import (
	"context"
	"fmt"
	"gotu/pkg/xlog"
	"gotu/pkg/xmsg"
	"gotu/pkg/xnet"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestUDP(t *testing.T) {
	ctx := context.Background()

	addr := ":8888"
	var wg sync.WaitGroup

	svr, err := xnet.NewUDPServer(ctx, xnet.UDPSvrArgs{
		Addr: addr,
		OnConnect: func(ctx context.Context, sock xnet.Socket) interface{} {
			return sock
		},
		OnDisconnect: func(ctx context.Context, state interface{}) {
			xlog.Get(ctx).Debug("Svr disconnect")
		},
		OnMsg: xmsg.ParseMsgWarp(func(ctx context.Context, arg xmsg.MsgArgs) error {
			defer wg.Done()
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

	defer func() {
		time.Sleep(1 * time.Second)
		svr.Close(ctx)
	}()

	cli, err := xnet.NewUDPClient(ctx, xnet.UDPCliArgs{
		Addr: addr,
		OnConnect: func(ctx context.Context, sock xnet.Socket) interface{} {
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

	for i := 0; i < 10; i++ {
		msg, err := xmsg.PackMsg(ctx, xmsg.PackMsgArgs{
			Payload: []byte(fmt.Sprintf("cli data %v", i)),
		})
		if err != nil {
			panic(err)
		}
		if err := cli.SendMsg(ctx, msg); err != nil {
			xlog.Get(ctx).Warn("Cli send msg failed.", zap.Any("err", err))
		} else {
			wg.Add(1)
		}
		time.Sleep(100 * time.Millisecond)

		if err := cli.Reconnect(ctx); err != nil {
			panic(err)
		}
	}

	wg.Wait()
	cli.Close(ctx)
}
