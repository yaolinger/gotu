package xregistry_test

import (
	"context"
	"gotu/pkg/xlog"
	"gotu/pkg/xmsg"
	"gotu/pkg/xnet"
	"gotu/pkg/xregistry"
	"testing"
	"time"

	"go.uber.org/zap"
)

func newServer(ctx context.Context) (*xnet.TCPServer, error) {
	// 服务端回调处理, xregistry.OnConnect， xregistry.OnDisconnect，xregistry.OnMsg
	return xnet.NewTCPServer(ctx, xnet.TCPSvrArgs{
		Addr:         ":9999",
		OnConnect:    xregistry.OnConnect,
		OnDisconnect: xregistry.OnDisconnect,
		OnMsg:        xmsg.ParseMsgWarp(xregistry.OnMsg),
	})

}

func newClient(ctx context.Context) (*xnet.TCPClient, error) {
	// 客户端仅测试使用 自定义回调处理, onConnect, onDisconnect, onMsg
	return xnet.NewTCPClient(ctx, xnet.TCPCliArgs{
		Addr: ":9999",
		OnConnect: func(ctx context.Context, sock xnet.Socket) interface{} {
			return nil
		},
		OnDisconnect: func(ctx context.Context, state interface{}) {
			xlog.Get(ctx).Debug("Cli disconnect")
		},
		OnMsg: xmsg.ParseMsgWarp(func(ctx context.Context, arg xmsg.MsgArgs) error {
			if arg.Header.Cmd == xregistry.CMD_ECHO {
				resp := &xregistry.EchoResp{}
				if err := xregistry.Unmarshal(arg.Payload, resp); err == nil {
					xlog.Get(ctx).Debug("Cli recv msg", zap.Int32("num", resp.Num))
				} else {
					xlog.Get(ctx).Warn("Unmarshal failed.", zap.Any("err", err))
				}
			}
			return nil
		}),
	})
}

func TestXregistry(t *testing.T) {
	ctx := context.Background()

	// 构建服务器
	svr, err := newServer(ctx)
	if err != nil {
		panic(err)
	}
	defer func() {
		time.Sleep(1 * time.Second)
		svr.Close(ctx)
	}()

	// 构建客户端
	cli, err := newClient(ctx)
	if err != nil {
		panic(err)
	}
	defer cli.Close(ctx)

	req := &xregistry.EchoReq{Num: 1024}
	// 序列化数据 payload
	payload, err := xregistry.Marshal(req)
	if err != nil {
		panic(err)
	}
	// 打包数据包 header + payload
	msg, err := xmsg.PackMsg(ctx, xmsg.PackMsgArgs{Cmd: xregistry.CMD_ECHO, Payload: payload})
	if err != nil {
		panic(err)
	}

	// 发送10个测试数据
	for i := 0; i < 10; i++ {
		if err := cli.SendMsg(ctx, msg); err != nil {
			panic(err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
