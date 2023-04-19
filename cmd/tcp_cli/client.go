package main

import (
	"context"
	"fmt"
	"gotu/pkg/xcommon"
	"gotu/pkg/xlog"
	"gotu/pkg/xmsg"
	"gotu/pkg/xnet"
	"strconv"
	"time"

	"go.uber.org/zap"
)

type ClientArgs struct {
	Num  int
	Addr string
}

type TcpClientState struct {
	delay int64
	count uint32
}

type TClient struct {
	arg *ClientArgs
}

func NewTClient(ctx context.Context, arg ClientArgs) *TClient {
	return &TClient{arg: &arg}
}

func (tcli *TClient) Start(ctx context.Context) {
	var wg xcommon.WaitGroup
	clis := []*xnet.TCPClient{}

	for i := 0; i < tcli.arg.Num; i++ {
		cli, err := xnet.NewTCPClient(ctx, xnet.TCPCliArgs{
			Addr: tcli.arg.Addr,
			OnConnect: func(ctx context.Context, sock xnet.Socket) interface{} {
				return &TcpClientState{}
			},
			OnDisconnect: func(ctx context.Context, state interface{}) {
				// s := state.(*TcpClientState)
				// if s.count != 0 {
				// 	//xlog.Get(ctx).Info("Client disconnect", zap.Any("average delay(ms)", s.delay/int64(s.count)))
				// }
			},
			OnMsg: xmsg.ParseMsgWarp(func(ctx context.Context, arg xmsg.MsgArgs) error {
				state := arg.State.(*TcpClientState)
				if n, err := strconv.Atoi(string(arg.Payload)); err == nil {
					state.delay += time.Now().UnixMilli() - int64(n)
					state.count += 1
				}
				return nil
			}),
		})
		if err != nil {
			xlog.Get(ctx).Warn("New tcp client failed.", zap.Any("err", err))
			continue
		}

		clis = append(clis, cli)

		// 降低并发创建压力
		time.Sleep(100 * time.Microsecond)

		wg.Add(1)
		go func() {
			defer wg.Done(ctx)
			for i := 0; i < 10000; i++ {
				msg, err := xmsg.PackMsg(ctx, xmsg.PackMsgArgs{
					Payload: []byte(fmt.Sprintf("%d", time.Now().UnixMilli())),
				})
				if err != nil {
					xlog.Get(ctx).Warn("Pack msg failed.", zap.Any("err", err))
					continue
				}
				if err := cli.SendMsg(ctx, msg); err != nil {
					xlog.Get(ctx).Warn("Cli send msg failed.", zap.Any("err", err))
				}
				time.Sleep(500 * time.Millisecond)
			}
			cli.Close(ctx)
		}()

	}
	xlog.Get(ctx).Info("Start success.", zap.Any("cli count", len(clis)))
	wg.Wait()
}
