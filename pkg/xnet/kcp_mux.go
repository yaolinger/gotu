package xnet

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"gonet/pkg/xlog"
	"io"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

var (
	kcpMuxHeaderSizeof = binary.Size(&kcpMuxHeader{})
	kcpExchangeSizeof  = binary.Size(&kcpExchange{})

	// 连接状态
	kmsSynSent     int32 = -2
	kmsListen      int32 = -1
	kmsSynRcvd     int32 = 0
	kmsEstablished int32 = 1 // 通信状态
	kmsFinWait1    int32 = 2
	kmsCloseWait   int32 = 3
	kmsFinWait2    int32 = 4
	kmsTimeWait    int32 = 5
	kmsLastAck     int32 = 6

	kcpInitTimeout  = 300 * time.Millisecond // 握手流程
	kcpCloseTimeout = 300 * time.Millisecond // 挥手流程
)

// 转换
func kmsString(kms int32) string {
	if kms == kmsSynSent {
		return "syn-sent"
	} else if kms == kmsListen {
		return "listen"
	} else if kms == kmsSynRcvd {
		return "syn-rcvd"
	} else if kms == kmsEstablished {
		return "established"
	} else if kms == kmsFinWait1 {
		return "fin-wait1"
	} else if kms == kmsFinWait2 {
		return "fin-wait2"
	} else if kms == kmsTimeWait {
		return "time-wait"
	} else if kms == kmsCloseWait {
		return "close-wait"
	} else if kms == kmsLastAck {
		return "last-ack"
	} else {
		return fmt.Sprintf("unknown-%v", kms)
	}
}

// 头包
type kcpMuxHeader struct {
	Inline bool // 是否为内置协议
}

// kcp 内置协议
type kcpExchange struct {
	State int32 // 状态
}

// 路由
type kcpMux struct {
	isInline bool

	state    int32
	isListen bool
	closeCh  chan struct{}
	initCh   chan struct{}

	handler OnHandlerOnce
}

func newKCPMux(handler OnHandlerOnce, isInline bool, isListen bool) *kcpMux {
	m := &kcpMux{isInline: isInline, isListen: isListen, closeCh: make(chan struct{}, 1), initCh: make(chan struct{}, 1), handler: handler}
	if isListen {
		m.state = kmsListen
	} else {
		m.state = kmsSynSent
	}
	return m
}

// 处理消息
func (mux *kcpMux) onMsg(ctx context.Context, sock *KCPSocket, state interface{}, msg []byte) (int, error) {
	// 未开启内置协议
	if !mux.isInline {
		return mux.handler(ctx, state, msg)
	}

	// 开启内置协议
	if len(msg) < kcpMuxHeaderSizeof {
		return 0, nil
	}
	header := &kcpMuxHeader{}
	ioReader := bytes.NewReader(msg[0:kcpMuxHeaderSizeof])
	if err := binary.Read(ioReader, binary.LittleEndian, header); err != nil {
		return 0, err
	}
	if !header.Inline {
		// 逻辑层协议
		if atomic.LoadInt32(&mux.state) == kmsEstablished {
			c, err := mux.handler(ctx, state, msg[kcpMuxHeaderSizeof:])
			if c != 0 {
				c += kcpMuxHeaderSizeof
			}
			return c, err
		} else {
			return 0, fmt.Errorf("state [%v] not established", kmsString(atomic.LoadInt32(&mux.state)))
		}
	} else {
		// 内置协议
		ke := &kcpExchange{}
		ioReader := bytes.NewReader(msg[kcpMuxHeaderSizeof : kcpMuxHeaderSizeof+kcpExchangeSizeof])
		if err := binary.Read(ioReader, binary.LittleEndian, ke); err != nil {
			return 0, err
		}
		return mux.inlineProtocol(ctx, sock, ke)
	}
}

func (mux *kcpMux) sendInline(ctx context.Context, sock *KCPSocket, state int32) {
	ke := &kcpExchange{State: state}
	ioWrite := bytes.NewBuffer(nil)

	err := func() error {
		err := binary.Write(ioWrite, binary.LittleEndian, ke)
		if err != nil {
			return err
		}
		return sock.send(ctx, true, ioWrite.Bytes())
	}()
	if err != nil {
		xlog.Get(ctx).Warn("Send close failed.", zap.Any("err", err))
	}
}

// 内置协议处理
func (mux *kcpMux) inlineProtocol(ctx context.Context, sock *KCPSocket, ke *kcpExchange) (int, error) {
	// 三次握手流程
	if ke.State == kmsSynSent && atomic.LoadInt32(&mux.state) == kmsListen {
		// 发起端 kmsSynSent 接收端 kmsListen
		atomic.StoreInt32(&mux.state, kmsSynRcvd)
		mux.sendInline(ctx, sock, kmsSynRcvd)
		return kcpMuxHeaderSizeof + kcpExchangeSizeof, nil
	} else if ke.State == kmsSynRcvd && atomic.LoadInt32(&mux.state) == kmsSynSent {
		// 发起端 kmsSynSent 接收端 kmsSynRcvd
		atomic.StoreInt32(&mux.state, kmsEstablished)
		mux.sendInline(ctx, sock, kmsEstablished)
		mux.initCh <- struct{}{}
		return kcpMuxHeaderSizeof + kcpExchangeSizeof, nil
	} else if ke.State == kmsEstablished && atomic.LoadInt32(&mux.state) == kmsSynRcvd {
		// 发起端 kmsEstablished 接收端 kmsSynRcvd
		atomic.StoreInt32(&mux.state, kmsEstablished)
		mux.initCh <- struct{}{}
		return kcpMuxHeaderSizeof + kcpExchangeSizeof, nil
	}

	// 四次挥手流程
	if ke.State == kmsFinWait1 && atomic.LoadInt32(&mux.state) == kmsEstablished {
		// 发起端 kmsFinWait1  接收端 kmsEstablished
		atomic.StoreInt32(&mux.state, kmsCloseWait)
		mux.sendInline(ctx, sock, kmsCloseWait)

		// 被动close
		atomic.StoreInt32(&mux.state, kmsLastAck)
		mux.sendInline(ctx, sock, kmsLastAck)
		return kcpMuxHeaderSizeof + kcpExchangeSizeof, nil
	} else if ke.State == kmsTimeWait && atomic.LoadInt32(&mux.state) == kmsLastAck {
		// 发起端 kmsTimeWait 接收端 kmsLastAck
		return kcpMuxHeaderSizeof + kcpExchangeSizeof, io.EOF
	} else if ke.State == kmsCloseWait && atomic.LoadInt32(&mux.state) == kmsFinWait1 {
		// 发起端 kmsFinWait2 接收端 kmsCloseWait
		atomic.StoreInt32(&mux.state, kmsFinWait2)
		return kcpMuxHeaderSizeof + kcpExchangeSizeof, nil
	} else if ke.State == kmsLastAck && atomic.LoadInt32(&mux.state) == kmsFinWait2 {
		// 发起端 kmsTimeWait 接收端 kmsLastAck
		atomic.StoreInt32(&mux.state, kmsTimeWait)
		mux.sendInline(ctx, sock, kmsTimeWait)
		mux.closeCh <- struct{}{}
		return kcpMuxHeaderSizeof + kcpExchangeSizeof, io.EOF
	}

	//TODO: 暂时未考虑双端一起close
	return 0, fmt.Errorf("inline invalid %v cur %v isServer %v", kmsString(ke.State), kmsString(atomic.LoadInt32(&mux.state)), mux.isListen)
}

// 发起握手
func (mux *kcpMux) init(ctx context.Context, sock *KCPSocket) error {
	// 未开启内置协议
	if !mux.isInline {
		return nil
	}
	if atomic.LoadInt32(&mux.state) == kmsSynSent {
		mux.sendInline(ctx, sock, kmsSynSent)

	}
	ticker := time.NewTicker(kcpInitTimeout)
	defer ticker.Stop()
	select {
	case <-ticker.C:
		return fmt.Errorf("mux init timeout %v", kmsString(atomic.LoadInt32(&mux.state)))
	case <-mux.initCh:
	}
	return nil
}

// 发起挥手
func (mux *kcpMux) close(ctx context.Context, sock *KCPSocket) {
	// 未开启内置协议
	if !mux.isInline {
		return
	}
	if atomic.LoadInt32(&mux.state) == kmsEstablished {
		atomic.StoreInt32(&mux.state, kmsFinWait1)
		mux.sendInline(ctx, sock, kmsFinWait1)
		ticker := time.NewTicker(kcpCloseTimeout)
		defer ticker.Stop()
		select {
		case <-ticker.C:
			xlog.Get(ctx).Info("mux close timeout", zap.Any("state", kmsString(atomic.LoadInt32(&mux.state))))
			return
		case <-mux.closeCh:
		}
	}
	xlog.Get(ctx).Info("mux close", zap.Any("state", kmsString(atomic.LoadInt32(&mux.state))))
}

// 补充kcp包头
func (mux *kcpMux) packMsg(inline bool, payload []byte) ([]byte, error) {
	// 未开启内置协议
	if !mux.isInline {
		return payload, nil
	}
	header := &kcpMuxHeader{Inline: inline}
	ioWrite := bytes.NewBuffer(nil)
	if err := binary.Write(ioWrite, binary.LittleEndian, header); err != nil {
		return nil, err
	}
	msg := ioWrite.Bytes()
	msg = append(msg, payload...)
	return msg, nil
}
