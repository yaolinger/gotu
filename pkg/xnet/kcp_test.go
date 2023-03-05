package xnet_test

import (
	"context"
	"gonet/pkg/xlog"
	"gonet/pkg/xnet"
	"testing"
	"time"

	"github.com/xtaci/kcp-go"
	"go.uber.org/zap"
)

func TestKCPServer(t *testing.T) {
	ctx := context.Background()
	data := []byte("buginventer")

	addr := ":9993"
	svr, err := xnet.NewKCPServer(ctx, xnet.KCPServerArgs{
		Addr:      addr,
		OnConnect: func(ctx context.Context, sock xnet.Socket) interface{} { return sock },
		OnDisconnect: func(ctx context.Context, state interface{}) {
			xlog.Get(ctx).Info("Cli disconnect")
		},
		OnMsg: func(ctx context.Context, state interface{}, msg []byte) (int, error) {
			xlog.Get(ctx).Info("Svr recv msg", zap.Any("msg", string(msg)))
			sock := state.(xnet.Socket)
			if err := sock.SendMsg(ctx, []byte("echo data")); err != nil {
				xlog.Get(ctx).Warn("Send svr msg failed.", zap.Any("err", err))
			}
			return len(msg), nil
		},
	})
	if err != nil {
		panic(err)
	}
	cli, err := xnet.NewKCPClient(ctx, xnet.KCPClientArgs{
		Addr:      addr,
		OnConnect: func(ctx context.Context, fs xnet.Socket) interface{} { return nil },
		OnDisconnect: func(ctx context.Context, state interface{}) {
			xlog.Get(ctx).Info("Svr disconnect")
		},
		OnMsg: func(ctx context.Context, state interface{}, msg []byte) (int, error) {
			xlog.Get(ctx).Info("Cli recv msg", zap.Any("msg", string(msg)))
			return len(msg), nil
		},
	})
	if err != nil {
		panic(err)
	}

	for i := 0; i < 10; i++ {
		err = cli.SendMsg(ctx, data)
		if err != nil {
			panic(err)
		}
		time.Sleep(10 * time.Microsecond)
	}

	time.Sleep(2 * time.Second)

	cli.Close(ctx)
	time.Sleep(2 * time.Second)
	svr.Close(ctx)
	xlog.Get(ctx).Info("KCP SNMP", zap.Any("data", kcp.DefaultSnmp.Copy()))
}
