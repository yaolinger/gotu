package xnet_test

import (
	"context"
	"gonet/pkg/xlog"
	"gonet/pkg/xmsg"
	"gonet/pkg/xnet"
	"sync"
	"testing"
	"time"

	"github.com/xtaci/kcp-go"
	"go.uber.org/zap"
)

// 串行关闭
func TestKCPServer(t *testing.T) {
	ctx := context.Background()
	var wg sync.WaitGroup

	addr := ":9993"
	svr, err := xnet.NewKCPServer(ctx, xnet.KCPServerArgs{
		Addr:      addr,
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

		}),
		IsInline: true,
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
		OnMsg: xmsg.ParseMsgWarp(func(ctx context.Context, arg xmsg.MsgArgs) error {
			defer wg.Done()
			xlog.Get(ctx).Info("Cli recv msg", zap.Any("msg", string(arg.Payload)))
			return nil
		}),
		IsInline: true,
	})
	if err != nil {
		panic(err)
	}

	data := []byte("buginventer")
	msg, err := xmsg.PackMsg(ctx, xmsg.PackMsgArgs{
		Payload: data,
	})
	if err != nil {
		panic(err)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		err = cli.SendMsg(ctx, msg)
		if err != nil {
			panic(err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	wg.Wait()

	cli.Close(ctx)
	time.Sleep(1 * time.Second)
	svr.Close(ctx)
	xlog.Get(ctx).Info("KCP SNMP", zap.Any("data", kcp.DefaultSnmp.Copy()))
}

// 并发关闭
func TestKCPServer2(t *testing.T) {
	ctx := context.Background()
	var wg sync.WaitGroup

	addr := ":9992"
	svr, err := xnet.NewKCPServer(ctx, xnet.KCPServerArgs{
		Addr:      addr,
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

		}),
		IsInline: true,
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
		OnMsg: xmsg.ParseMsgWarp(func(ctx context.Context, arg xmsg.MsgArgs) error {
			defer wg.Done()
			xlog.Get(ctx).Info("Cli recv msg", zap.Any("msg", string(arg.Payload)))
			return nil
		}),
		IsInline: true,
	})
	if err != nil {
		panic(err)
	}

	data := []byte("buginventer")
	msg, err := xmsg.PackMsg(ctx, xmsg.PackMsgArgs{
		Payload: data,
	})
	if err != nil {
		panic(err)
	}

	for i := 0; i < 10; i++ {
		wg.Add(1)
		err = cli.SendMsg(ctx, msg)
		if err != nil {
			panic(err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	wg.Wait()

	wg.Add(1)
	go func() {
		cli.Close(ctx)
		wg.Done()
	}()
	svr.Close(ctx)
	wg.Wait()
	xlog.Get(ctx).Info("KCP SNMP", zap.Any("data", kcp.DefaultSnmp.Copy()))
}
