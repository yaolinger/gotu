package handlers

import (
	"context"
	"fmt"
	"gotu/pkg/xactor"
	"gotu/pkg/xlatency"
	"gotu/pkg/xlog"
	"gotu/pkg/xnet"

	"go.uber.org/zap"
)

type Registry struct {
	proxyAddr string
	mode      bool
}

var reg = &Registry{}

func InitRegistry(addr string, mode bool) *Registry {
	reg.proxyAddr = addr
	reg.mode = mode
	return reg
}

func (r *Registry) OnConnect(ctx context.Context, sock xnet.Socket) interface{} {
	s := &State{
		svrSock: sock,
	}

	if l, err := xlatency.NewLatencyActor(ctx, xlatency.LatencyMockArgs{Name: fmt.Sprintf("latencyActor-%v", sock.RemoteAddr()), Mode: r.mode, Loss: 20, Latency: 50}); err != nil {
		xlog.Get(ctx).Warn("New latency failed.", zap.Any("err", err))
		return s
	} else {
		s.latency = l
	}

	cli, err := xnet.NewUDPClient(ctx, xnet.UDPCliArgs{
		Addr:    reg.proxyAddr,
		Timeout: 10,
		OnConnect: func(ctx context.Context, sock xnet.Socket) interface{} {
			xlog.Get(ctx).Sugar().Debugf("Proxy connect success, %v => %v", s.svrSock.RemoteAddr(), sock.RemoteAddr())
			return sock
		},
		OnDisconnect: func(ctx context.Context, state interface{}) {
			xlog.Get(ctx).Sugar().Debugf("Proxy disconnect %v => %v", s.svrSock.RemoteAddr(), sock.RemoteAddr())
		},
		OnMsg: func(ctx context.Context, state interface{}, msg []byte) (int, error) {
			xactor.AsyncRequest(ctx, s.latency.Name(), &xlatency.RecvFromCliReq{
				Msg: msg,
			})
			return 0, nil
		},
	})
	if err != nil {
		xlog.Get(ctx).Warn("New udp cli failed.", zap.Any("err", err))
		return s
	}
	s.cli = cli

	if _, err := xactor.SyncRequest[xlatency.RegisterSendToCliReq, xlatency.RegisterSendToCliResp](ctx, s.latency.Name(), &xlatency.RegisterSendToCliReq{
		SendToCli: func(ctx context.Context, b []byte) {
			if err := s.cli.SendMsg(ctx, b); err != nil {
				xlog.Get(ctx).Warn("Send to client failed.", zap.Any("err", err))
			}
		},
	}); err != nil {
		xlog.Get(ctx).Warn("Register send-to-cli failed.", zap.Any("err", err))
	}

	if _, err := xactor.SyncRequest[xlatency.RegisterSendToSvrReq, xlatency.RegisterSendToSvrResp](ctx, s.latency.Name(), &xlatency.RegisterSendToSvrReq{
		SendToSvr: func(ctx context.Context, b []byte) {
			if err := sock.SendMsg(ctx, b); err != nil {
				xlog.Get(ctx).Warn("Send to svr failed.", zap.Any("err", err))
			}
		},
	}); err != nil {
		xlog.Get(ctx).Warn("Register send-to-svr failed.", zap.Any("err", err))
	}
	return s
}

func (r *Registry) OnDisconnect(ctx context.Context, state interface{}) {
	s := state.(*State)
	s.cli.Close(ctx)

	if actor, err := xactor.GetActor(s.latency.Name()); err == nil {
		actor.Close(ctx)
	} else {
		xlog.Get(ctx).Warn("Get actor failed.", zap.Any("name", s.latency.Name()))
	}
}

func (r *Registry) OnMsg(ctx context.Context, state interface{}, msg []byte) (int, error) {
	s := state.(*State)
	xactor.AsyncRequest(ctx, s.latency.Name(), &xlatency.RecvFromSvrReq{Msg: msg})
	return 0, nil
}
