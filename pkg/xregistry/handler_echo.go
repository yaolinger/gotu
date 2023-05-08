package xregistry

import (
	"context"
	"gotu/pkg/xlog"

	"go.uber.org/zap"
)

func init() {
	// 注册协议
	Register(CMD_ECHO, HandleWarp(Echo))
}

var CMD_ECHO = int32(1)

type EchoReq struct {
	Num int32
}

type EchoResp struct {
	Num int32
}

// handler echo
func Echo(ctx context.Context, state *State, req *EchoReq) error {
	xlog.Get(ctx).Info("Svr recv echo", zap.Int32("num", req.Num), zap.String("addr", state.Sock.RemoteAddr().String()))
	state.SendMsg(ctx, CMD_ECHO, &EchoResp{Num: req.Num})
	return nil
}
