package xlatency

import (
	"context"
	"gotu/pkg/xactor"
	"gotu/pkg/xcommon"
	"math/rand"
	"time"
)

type latencyMsg struct {
	inout bool // in 入站, out 出站
	at    int64
	msg   []byte
}

type LatencyMockArgs struct {
	Name       string
	InLoss     uint32 // 入站丢失率 0~100
	InLatency  uint32 // 入站延迟ms
	OutLoss    uint32 // 入站
	OutLatency uint32 // 入站
}

// 延迟模拟模块 => 针对于udp网络
type LatencyActor struct {
	name       string
	inLoss     uint32
	inLatency  uint32
	outLoss    uint32
	outLatency uint32

	msgs []*latencyMsg

	inBytes       uint32
	inLostBytes   uint32
	inLostPackets uint32
	inPackets     uint32
	inDelay       int64

	outBytes       uint32
	outLostBytes   uint32
	outLostPackets uint32
	outPackets     uint32
	outDelay       int64

	sendToCli func(context.Context, []byte)
	sendToSvr func(context.Context, []byte)
}

func NewLatencyActor(ctx context.Context, arg LatencyMockArgs) (*LatencyActor, error) {
	l := &LatencyActor{
		name:       arg.Name,
		inLoss:     arg.InLoss,
		inLatency:  arg.InLatency,
		outLoss:    arg.OutLoss,
		outLatency: arg.OutLatency,
		msgs:       make([]*latencyMsg, 0),
	}
	if err := xactor.NewActorGroutine(ctx, l); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *LatencyActor) InitArg() xactor.ActorHandlerArgs {
	return xactor.ActorHandlerArgs{
		Syncs:          []xactor.SyncHandlerArgs{xactor.SyncHandlerWrap(l.RegisterSendToCli), xactor.SyncHandlerWrap(l.RegisterSendToSvr)},
		Asyncs:         []xactor.AsyncHandlerArgs{xactor.AsyncHandlerWrap(l.RecvFromCli), xactor.AsyncHandlerWrap(l.RecvFromSvr)},
		Tickers:        []xactor.TickHanler{l.tickLoop},
		TickerDuration: 1 * time.Millisecond,
	}
}

func (l *LatencyActor) Name() string {
	return l.name
}

func (l *LatencyActor) Close(ctx context.Context) {

	var inAverageDelay uint64
	inTransfer := l.inPackets - l.inLostPackets
	if inTransfer != 0 {
		inAverageDelay = uint64(l.inDelay) / uint64(inTransfer)
	}
	var outAverageDelay uint64
	outTransfer := l.outPackets - l.outLostPackets
	if outTransfer != 0 {
		outAverageDelay = uint64(l.outDelay) / uint64(outTransfer)
	}

	// 输出统计数据
	xcommon.PrintTable(ctx, []string{"latency-name", "bytes(B)", "lost-bytes(B)", "packets", "lost-packets", "all-delay(ms)", "average-delay(ms)"},
		[][]string{{l.name + "(in)", xcommon.ToString(l.inBytes), xcommon.ToString(l.inLostBytes), xcommon.ToString(l.inPackets), xcommon.ToString(l.inLostPackets), xcommon.ToString(l.inDelay), xcommon.ToString(inAverageDelay)},
			{l.name + "(out)", xcommon.ToString(l.outBytes), xcommon.ToString(l.outLostBytes), xcommon.ToString(l.outPackets), xcommon.ToString(l.outLostPackets), xcommon.ToString(l.outDelay), xcommon.ToString(outAverageDelay)}})
}

func (l *LatencyActor) isLoss(inout bool, len int) bool {
	if inout {
		l.inPackets++
		rand.Seed(time.Now().UnixMilli())
		if rand.Int31n(100) < int32(l.inLoss) {
			l.inLostPackets++
			l.inLostBytes += uint32(len)
			return true
		}
	} else {
		l.outPackets++
		rand.Seed(time.Now().UnixMilli())
		if rand.Int31n(100) < int32(l.outLoss) {
			l.outLostPackets++
			l.outLostBytes += uint32(len)
			return true
		}
	}
	return false
}

func (l *LatencyActor) randLatency(inout bool) int64 {
	if inout {
		if l.inLatency > 0 {
			rl := int64(rand.Int31n(int32(l.inLatency)))
			l.inDelay += rl
			return rl
		}
	} else {
		if l.outLatency > 0 {
			rl := int64(rand.Int31n(int32(l.outLatency)))
			l.outDelay += rl
			return rl
		}
	}
	return 0
}

func (l *LatencyActor) extend(ctx context.Context, inout bool, msg []byte) {
	if inout {
		l.inBytes += uint32(len(msg))
	} else {
		l.outBytes += uint32(len(msg))
	}
	l.msgs = append(l.msgs, &latencyMsg{inout: inout, msg: msg, at: time.Now().UnixMilli() + l.randLatency(inout)})
}

type RegisterSendToSvrReq struct {
	SendToSvr func(context.Context, []byte)
}

type RegisterSendToSvrResp struct {
}

// 同步注册 sendToSvr
func (l *LatencyActor) RegisterSendToSvr(ctx context.Context, req *RegisterSendToSvrReq) (*RegisterSendToSvrResp, error) {
	l.sendToSvr = req.SendToSvr
	return &RegisterSendToSvrResp{}, nil
}

type RegisterSendToCliReq struct {
	SendToCli func(context.Context, []byte)
}

type RegisterSendToCliResp struct {
}

// 同步注册 sendToCli
func (l *LatencyActor) RegisterSendToCli(ctx context.Context, req *RegisterSendToCliReq) (*RegisterSendToCliResp, error) {
	l.sendToCli = req.SendToCli
	return &RegisterSendToCliResp{}, nil
}

type RecvFromCliReq struct {
	Msg []byte
}

// 异步接收client数据
func (l *LatencyActor) RecvFromCli(ctx context.Context, req *RecvFromCliReq) {
	if l.isLoss(false, len(req.Msg)) {
		return
	}

	l.extend(ctx, false, req.Msg)
}

type RecvFromSvrReq struct {
	Msg []byte
}

// 异步接收server数据
func (l *LatencyActor) RecvFromSvr(ctx context.Context, req *RecvFromSvrReq) {
	if l.isLoss(true, len(req.Msg)) {
		return
	}

	l.extend(ctx, true, req.Msg)
}

func (l *LatencyActor) tickLoop(ctx context.Context) {
	now := time.Now().UnixMilli()
	msgs := make([]*latencyMsg, 0)
	for _, m := range l.msgs {
		if m.at > now {
			msgs = append(msgs, m)
		} else {
			if m.inout && l.sendToCli != nil {
				l.sendToCli(ctx, m.msg)
			} else if !m.inout && l.sendToSvr != nil {
				l.sendToSvr(ctx, m.msg)
			}
		}
	}
	l.msgs = msgs
}
