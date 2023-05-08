package xregistry

import (
	"context"
	"gotu/pkg/xlog"
	"gotu/pkg/xmsg"
	"gotu/pkg/xnet"

	"go.uber.org/zap"
)

// State: 逻辑层state, 对应一条网络连接,可附加一些状态信息(如, uid, token), xnet conn => State
type State struct {
	Sock xnet.Socket

	// UID
	// TOKEN
}

func (s *State) SendMsg(ctx context.Context, cmd int32, data interface{}) {
	// 序列化数据 payload
	payload, err := Marshal(data)
	if err != nil {
		xlog.Get(ctx).Warn("Marshal data failed.", zap.Any("err", err), zap.Int32("cmd", cmd), zap.Any("data", data))
		return
	}
	// 打包数据包 header + payload
	msg, err := xmsg.PackMsg(ctx, xmsg.PackMsgArgs{Cmd: cmd, Payload: payload})
	if err != nil {
		xlog.Get(ctx).Warn("Pack msg failed.", zap.Any("err", err), zap.Int32("cmd", cmd), zap.Any("data", data))
		return
	}
	// 发送数据
	if err := s.Sock.SendMsg(ctx, msg); err != nil {
		xlog.Get(ctx).Warn("Send msg failed.", zap.Any("err", err), zap.Int32("cmd", cmd))
	}
}
