package xnet

import (
	"context"
	"time"
)

const (
	tcpNetwork     = "tcp"
	readBufferSize = 1024

	writeTimeout = 10 * time.Second // 写超时时间
	readTimeout  = 10 * time.Second // 读超时时间

	writeChanLimit = 200 // 写channel大小

	kcpSocketStart = 0 // kcp socket 开启
	kcpSocketClose = 1 // kcp socket 关闭

	kcpAckNoDelay = true // 非延迟ack
	kcpNoDelay    = 1    // 1:RTO=30ms,0:RTO=100ms
	kcpInterval   = 20   // 工作间隔(越小越块,cpu越高, 10, 20, 30, 40)
	kcpResend     = 2    // 快速重传(0关闭)
	kcpNC         = 1    // 是否关闭流空(0:开启,1:关闭)
)

// 消息处理
type OnHandlerOnce func(ctx context.Context, state interface{}, msg []byte) (int, error)

// 建立链接
type OnConnect func(ctx context.Context, sock Socket) interface{}

// 关闭链接
type OnDisconnect func(ctx context.Context, state interface{})

type Socket interface {
	SendMsg(ctx context.Context, msg []byte) error
}
