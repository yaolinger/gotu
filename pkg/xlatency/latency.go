package xlatency

import (
	"context"
	"gotu/pkg/xlog"
	"math/rand"
	"sync"
	"time"
)

type LatencyMockArgs struct {
	Loss uint32 // 丢失率 0~100
}

// 延迟模拟模块 => 针对于udp网络
type LatencyMock struct {
	loss uint32

	mu        sync.Mutex
	sendToCli func(context.Context, []byte)
	sendToSvr func(context.Context, []byte)
}

func NewLatencyMock(ctx context.Context, arg LatencyMockArgs) *LatencyMock {
	l := &LatencyMock{
		loss: arg.Loss,
	}
	return l
}

func (l *LatencyMock) isLoss() bool {
	rand.Seed(time.Now().UnixMilli())
	return rand.Int31n(100) < int32(l.loss)
}

func (l *LatencyMock) SetSendToSvr(sendToSvr func(context.Context, []byte)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sendToSvr = sendToSvr
}

func (l *LatencyMock) SetSendToCli(sendToCli func(context.Context, []byte)) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.sendToCli = sendToCli
}

func (l *LatencyMock) RecvFromCli(ctx context.Context, mode bool, msg []byte) {
	if l.isLoss() {
		xlog.Get(ctx).Debug("packet loss")
		return
	}

	msg = l.extend(ctx, mode, msg)

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.sendToSvr != nil {
		l.sendToSvr(ctx, msg)
	}
}

func (l *LatencyMock) RecvFromSvr(ctx context.Context, mode bool, msg []byte) {
	if l.isLoss() {
		xlog.Get(ctx).Debug("packet loss")
		return
	}

	msg = l.extend(ctx, mode, msg)

	l.mu.Lock()
	defer l.mu.Unlock()
	if l.sendToCli != nil {
		l.sendToCli(ctx, msg)
	}
}

func (l *LatencyMock) extend(ctx context.Context, mode bool, msg []byte) []byte {
	return msg
}
