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

)

// 消息处理
type OnHandlerOnce func(ctx context.Context, state interface{}, msg []byte) (int, error)

// 建立链接
type OnConnect func(ctx context.Context, sock *TCPSocket) interface{}

// 关闭链接
type OnDisconnect func(ctx context.Context, state interface{})
