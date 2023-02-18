package xnet_test

import (
	"context"
	"fmt"
	"gonet/pkg/xlog"
	"gonet/pkg/xnet"
	"testing"
	"time"

	"go.uber.org/zap"
)

type State struct {
	sock *xnet.TCPSocket
}

func TestTCP(t *testing.T) {
	ctx := context.Background()

	// TODO 未做分包处理

	svr, err := xnet.NewTCPServer(ctx, xnet.TCPSvrArgs{
		Addr: ":9999",
		OnConnect: func(ctx context.Context, sock *xnet.TCPSocket) interface{} {
			return &State{sock: sock}
		},
		OnDisconnect: func(ctx context.Context, state interface{}) {
			xlog.Get(ctx).Debug("Svr disconnect")
		},
		OnMsg: func(ctx context.Context, state interface{}, msg []byte) (int, error) {
			if len(msg) == 0 {
				return 0, nil
			}
			xlog.Get(ctx).Debug("Svr recv msg", zap.String("msg", string(msg)))
			s := state.(*State)
			if err := s.sock.SendMsg(ctx, []byte("svr data")); err != nil {
				xlog.Get(ctx).Warn("Svr send msg failed.", zap.Any("err", err))
			}

			return len(msg), nil
		},
	})

	if err != nil {
		panic(err)
	}

	defer func() {
		time.Sleep(1 * time.Second)
		svr.Close(ctx)
	}()

	cli, err := xnet.NewTCPClient(ctx, xnet.TCPCliArgs{
		Addr: ":9999",
		OnConnect: func(ctx context.Context, sock *xnet.TCPSocket) interface{} {
			return nil
		},
		OnDisconnect: func(ctx context.Context, state interface{}) {
			xlog.Get(ctx).Debug("Cli disconnect")
		},
		OnMsg: func(ctx context.Context, state interface{}, msg []byte) (int, error) {
			if len(msg) == 0 {
				return 0, nil
			}
			xlog.Get(ctx).Debug("Cli recv msg", zap.String("msg", string(msg)))
			return len(msg), nil
		},
	})
	if err != nil {
		panic(err)
	}

	for i := 0; i < 10; i++ {
		if err := cli.SendMsg(ctx, []byte(fmt.Sprintf("cli data %v", i))); err != nil {
			xlog.Get(ctx).Warn("Cli send msg failed.", zap.Any("err", err))
		}
		time.Sleep(100 * time.Millisecond)
	}
	cli.Close(ctx)
}
