package handlers

import (
	"context"
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
		latency: xlatency.NewLatencyMock(ctx, xlatency.LatencyMockArgs{Loss: 20}),
		svrSock: sock,
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
			s.latency.RecvFromCli(ctx, r.mode, msg)
			return 0, nil
		},
	})
	if err != nil {
		xlog.Get(ctx).Warn("New udp cli failed.", zap.Any("err", err))
		return s
	}
	s.cli = cli
	s.latency.SetSendToCli(func(ctx context.Context, b []byte) {
		if err := s.cli.SendMsg(ctx, b); err != nil {
			xlog.Get(ctx).Warn("Send to client failed.", zap.Any("err", err))
		}
	})
	s.latency.SetSendToSvr(func(ctx context.Context, b []byte) {
		if err := sock.SendMsg(ctx, b); err != nil {
			xlog.Get(ctx).Warn("Send to svr failed.", zap.Any("err", err))
		}
	})
	return s
}

func (r *Registry) OnDisconnect(ctx context.Context, state interface{}) {
	s := state.(*State)
	s.cli.Close(ctx)
}

func (r *Registry) OnMsg(ctx context.Context, state interface{}, msg []byte) (int, error) {
	s := state.(*State)
	s.latency.RecvFromSvr(ctx, reg.mode, msg)
	return 0, nil
}
