package xlatency

import (
	"context"
	"fmt"
	"gotu/pkg/xactor"
	"gotu/pkg/xlog"
	"math/rand"
	"time"
)

type latencyMsg struct {
	isSvr bool
	at    int64
	msg   []byte
}

type LatencyMockArgs struct {
	Name    string
	Mode    int
	Loss    uint32 // 丢失率 0~100
	Latency uint32 // 延迟ms
}

// 延迟模拟模块 => 针对于udp网络
type LatencyActor struct {
	name string
	mode int

	loss    uint32
	latency uint32
	msgs    []*latencyMsg

	lost     uint32
	packets  uint32
	allDelay int64

	sendToCli func(context.Context, []byte)
	sendToSvr func(context.Context, []byte)
}

func NewLatencyActor(ctx context.Context, arg LatencyMockArgs) (*LatencyActor, error) {
	l := &LatencyActor{
		name:     arg.Name,
		mode:     arg.Mode,
		loss:     arg.Loss,
		latency:  arg.Latency,
		msgs:     make([]*latencyMsg, 0),
		lost:     0,
		packets:  0,
		allDelay: 0,
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
		TickerDuration: 5 * time.Millisecond,
	}
}

func (l *LatencyActor) Name() string {
	return l.name
}

func (l *LatencyActor) Close(ctx context.Context) {

	var averageDelay uint64
	transfer := l.packets - l.lost
	if transfer != 0 {
		averageDelay = uint64(l.allDelay) / uint64(transfer)
	}

	record := fmt.Sprintf("Latency[%v] all packets[%v] lost packets[%v] allDelay[%vms] averageDelay[%vms]", l.name, l.packets, l.lost, l.allDelay, averageDelay)

	xlog.Get(ctx).Info(record)
}

func (l *LatencyActor) isLoss() bool {
	l.packets++
	rand.Seed(time.Now().UnixMilli())
	if rand.Int31n(100) < int32(l.loss) {
		l.lost++
		return true
	}
	return false
}

func (l *LatencyActor) randLatency() int64 {
	if l.latency > 0 {
		rl := int64(rand.Int31n(int32(l.latency)))
		l.allDelay += rl
		return rl
	}
	return 0
}

func (l *LatencyActor) extend(ctx context.Context, isSvr bool, msg []byte) {
	l.msgs = append(l.msgs, &latencyMsg{isSvr: isSvr, msg: msg, at: time.Now().UnixMilli() + l.randLatency()})
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
	if l.isLoss() {
		return
	}

	l.extend(ctx, true, req.Msg)
}

type RecvFromSvrReq struct {
	Msg []byte
}

// 异步接收server数据
func (l *LatencyActor) RecvFromSvr(ctx context.Context, req *RecvFromSvrReq) {
	if l.isLoss() {
		return
	}

	l.extend(ctx, false, req.Msg)
}

func (l *LatencyActor) tickLoop(ctx context.Context) {
	now := time.Now().UnixMilli()
	msgs := make([]*latencyMsg, 0)
	for _, m := range l.msgs {
		if m.at > now {
			msgs = append(msgs, m)
		} else {
			if m.isSvr && l.sendToSvr != nil {
				l.sendToSvr(ctx, m.msg)
			} else if !m.isSvr && l.sendToCli != nil {
				l.sendToCli(ctx, m.msg)
			}
		}
	}
	l.msgs = msgs
}
