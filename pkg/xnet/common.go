package xnet

import (
	"context"
	"fmt"
	"net"
	"time"
)

const (
	tcpNetwork     = "tcp"
	udpNetwork     = "udp"
	readBufferSize = 1024

	writeTimeout = 10 * time.Second // 写超时时间
	readTimeout  = 60 * time.Second // 读超时时间

	writeChanLimit = 200 // 写channel大小

	kcpSocketStart = 0 // kcp socket 开启
	kcpSocketClose = 1 // kcp socket 关闭

	kcpAckNoDelay = true // 非延迟ack
	kcpNoDelay    = 1    // 1:RTO=30ms,0:RTO=100ms
	kcpInterval   = 20   // 工作间隔(越小越块,cpu越高, 10, 20, 30, 40)
	kcpResend     = 2    // 快速重传(0关闭)
	kcpNC         = 1    // 是否关闭拥塞算法(0:开启,1:关闭)

	// 所罗门编码: 15个数据包接收到任意10个数据包都可以恢复全部源数据包
	kcpFecDataShards   = 10 // fec源数据包数量
	kcpFecParityShards = 5  // fec生成数据包数量

	maxMessageSize = 1024 * 2 // Websocket请求包大小上限

	udpCheckDuration  = 3 * time.Second // 检查时钟
	udpSessionTimeout = 10              // udp超时(s)
	udpMsgChanLimit   = 1024            // msg channel 带线啊哦
)

// 消息处理
type OnHandlerOnce func(ctx context.Context, state interface{}, msg []byte) (int, error)

// 建立链接
type OnConnect func(ctx context.Context, sock Socket) interface{}

// 关闭链接
type OnDisconnect func(ctx context.Context, state interface{})

type Socket interface {
	SendMsg(ctx context.Context, msg []byte) error
	RemoteAddr() net.Addr
	LocalAddr() net.Addr
}

// udp 消息处理
type udpOnMsg func(context.Context, []byte, *net.UDPAddr)

// udp 发送消息
type udpSendMsg func(ctx context.Context, datagram *udpDatagram) error

// 地址转换
func addrToString(addr *net.UDPAddr) string {
	return fmt.Sprintf("%s:%v", addr.IP.String(), addr.Port)
}
