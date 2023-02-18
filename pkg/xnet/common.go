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
)

// 消息处理
type OnHandlerOnce func(ctx context.Context, msg []byte) (int, error)
