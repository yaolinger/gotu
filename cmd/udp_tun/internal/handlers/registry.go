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

const (
	modeNormal = 0
	modeClient = 1
	modeServer = 2
)

func modeString(mode int) string {
	if mode == modeNormal {
		return "normal"
	} else if mode == modeClient {
		return "client"
	} else if mode == modeServer {
		return "server"
	} else {
		return fmt.Sprintf("mode[%v] invalid", mode)
	}
}

type RegistryArgs struct {
	Addr    string
	Mode    int
	Header  bool
	Loss    uint32
	Latency uint32
}

type Registry struct {
	proxyAddr string
	mode      int
	header    bool
	loss      uint32
	latency   uint32
}

var reg = &Registry{}

func InitRegistry(ctx context.Context, arg RegistryArgs) (*Registry, error) {
	if arg.Mode != modeNormal && arg.Mode != modeClient && arg.Mode != modeServer {
		return nil, fmt.Errorf("mode[%v] invalid[0|1|2]", arg.Mode)
	}

	xlog.Get(ctx).Sugar().Debugf("Mode [%v] proxy[%v] header[%v] loss[%v] latency[%v] start success.", modeString(arg.Mode), arg.Addr, arg.Header, arg.Loss, arg.Latency)

	reg.proxyAddr = arg.Addr
	reg.mode = arg.Mode
	reg.header = arg.Header
	reg.loss = arg.Loss
	reg.latency = arg.Latency
	return reg, nil
}

// TODO 存在一些吞错误的行为, 考虑重新设计流程
func (r *Registry) OnConnect(ctx context.Context, sock xnet.Socket) interface{} {
	s := &State{
		svrSock: sock,
		snmp:    getSingleTunSnmp(fmt.Sprintf("%v", sock.RemoteAddr())),
	}

	if l, err := xlatency.NewLatencyActor(ctx, xlatency.LatencyMockArgs{Name: fmt.Sprintf("latencyActor-%v", sock.RemoteAddr()), Mode: r.mode, Loss: r.loss, Latency: r.latency}); err != nil {
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
			// client/server proxy add header
			var err error
			if r.mode == modeClient && r.header {
				var delay int64
				delay, msg, err = unpack(ctx, msg)
				if err != nil {
					return 0, err
				}
				s.snmp.recv(len(msg), delay)
			} else if r.mode == modeServer && r.header {
				msg, err = pack(ctx, msg)
				if err != nil {
					return 0, err
				}
				s.snmp.send(len(msg))
			}
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
	s.snmp.close(ctx)

	if actor, err := xactor.GetActor(s.latency.Name()); err == nil {
		actor.Close(ctx)
	} else {
		xlog.Get(ctx).Warn("Get actor failed.", zap.Any("name", s.latency.Name()))
	}
}

func (r *Registry) OnMsg(ctx context.Context, state interface{}, msg []byte) (int, error) {
	s := state.(*State)

	// client/server proxy add header
	var err error
	if r.mode == modeClient && r.header {
		msg, err = pack(ctx, msg)
		if err != nil {
			return 0, err
		}
		s.snmp.send(len(msg))
	} else if r.mode == modeServer && r.header {
		var delay int64
		delay, msg, err = unpack(ctx, msg)
		if err != nil {
			return 0, err
		}
		s.snmp.recv(len(msg), delay)
	}

	xactor.AsyncRequest(ctx, s.latency.Name(), &xlatency.RecvFromSvrReq{Msg: msg})
	return 0, nil
}
