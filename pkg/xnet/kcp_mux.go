package xnet

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"gotu/pkg/xlog"
	"io"
	"sync"
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
	kmsClosing     int32 = 5
	kmsTimeWait    int32 = 6
	kmsLastAck     int32 = 7

	kcpInitTimeout  = 300 * time.Millisecond // 握手流程
	kcpCloseTimeout = 300 * time.Millisecond // 挥手流程

	kcpMuxUninit int32 = 0 // 未初始化
	kcpMuxInit   int32 = 1 // 初始化
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
	} else if kms == kmsClosing {
		return "closing"
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

	initFlag  int32
	closeOnce sync.Once

	state    int32
	isListen bool
	closeCh  chan struct{}
	initCh   chan struct{}

	handler OnHandlerOnce
}

func newKCPMux(handler OnHandlerOnce, isInline bool, isListen bool) *kcpMux {
	m := &kcpMux{isInline: isInline,
		isListen: isListen,
		initFlag: kcpMuxUninit,
		closeCh:  make(chan struct{}, 1),
		initCh:   make(chan struct{}, 1),
		handler:  handler}
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
	if ke.State == kmsSynSent && atomic.CompareAndSwapInt32(&mux.state, kmsListen, kmsSynRcvd) {
		// 主动发起端 kmsSynSent 被动接收端 kmsListen
		mux.sendInline(ctx, sock, kmsSynRcvd)
		return kcpMuxHeaderSizeof + kcpExchangeSizeof, nil
	} else if ke.State == kmsSynRcvd && atomic.CompareAndSwapInt32(&mux.state, kmsSynSent, kmsEstablished) {
		// 被动发起端 kmsSynSent 主动接收端 kmsSynRcvd
		mux.sendInline(ctx, sock, kmsEstablished)
		mux.initCh <- struct{}{}
		return kcpMuxHeaderSizeof + kcpExchangeSizeof, nil
	} else if ke.State == kmsEstablished && atomic.CompareAndSwapInt32(&mux.state, kmsSynRcvd, kmsEstablished) {
		// 主动发起端 kmsEstablished 被动接收端 kmsSynRcvd
		mux.initCh <- struct{}{}
		return kcpMuxHeaderSizeof + kcpExchangeSizeof, nil
	}

	// 四次挥手流程
	if ke.State == kmsFinWait1 && atomic.CompareAndSwapInt32(&mux.state, kmsEstablished, kmsCloseWait) {
		// 主动发起端 kmsFinWait1  被动接收端 kmsEstablished
		mux.sendInline(ctx, sock, kmsCloseWait)
		// 被动close
		if atomic.CompareAndSwapInt32(&mux.state, kmsCloseWait, kmsLastAck) {
			mux.sendInline(ctx, sock, kmsLastAck)
		}
		return kcpMuxHeaderSizeof + kcpExchangeSizeof, nil
	} else if ke.State == kmsTimeWait && atomic.LoadInt32(&mux.state) == kmsLastAck {
		// 主动发起端 kmsTimeWait 被动接收端 kmsLastAck
		mux.closeCh <- struct{}{}
		return kcpMuxHeaderSizeof + kcpExchangeSizeof, io.EOF
	} else if ke.State == kmsCloseWait && atomic.CompareAndSwapInt32(&mux.state, kmsFinWait1, kmsFinWait2) {
		// 被动发起端 kmsCloseWait 主动接收端 kmsFinWait1
		return kcpMuxHeaderSizeof + kcpExchangeSizeof, nil
	} else if ke.State == kmsLastAck && atomic.CompareAndSwapInt32(&mux.state, kmsFinWait2, kmsTimeWait) {
		// 被动发起端 kmsLastAck 主动接收端 kmsFinWait2
		mux.sendInline(ctx, sock, kmsTimeWait)
		mux.closeCh <- struct{}{}
		return kcpMuxHeaderSizeof + kcpExchangeSizeof, io.EOF
	}

	// 双端同时close
	if ke.State == kmsFinWait1 && atomic.CompareAndSwapInt32(&mux.state, kmsFinWait1, kmsClosing) {
		// 同时进入closing状态
		mux.sendInline(ctx, sock, kmsClosing)
		return kcpMuxHeaderSizeof + kcpExchangeSizeof, nil
	} else if ke.State == kmsClosing && atomic.CompareAndSwapInt32(&mux.state, kmsClosing, kmsTimeWait) {
		// 同时进入timewait状态
		mux.sendInline(ctx, sock, kmsTimeWait)
		return kcpMuxHeaderSizeof + kcpExchangeSizeof, nil
	} else if ke.State == kmsTimeWait && atomic.LoadInt32(&mux.state) == kmsTimeWait {
		// 接收到 timewait 触发关闭
		mux.closeCh <- struct{}{}
		return kcpMuxHeaderSizeof + kcpExchangeSizeof, io.EOF
	}

	return 0, fmt.Errorf("inline invalid %v cur %v isServer %v", kmsString(ke.State), kmsString(atomic.LoadInt32(&mux.state)), mux.isListen)
}

// 发起握手
func (mux *kcpMux) init(ctx context.Context, sock *KCPSocket) error {
	if atomic.CompareAndSwapInt32(&mux.initFlag, kcpMuxUninit, kcpMuxInit) {
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
	}
	return nil
}

// 发起挥手
func (mux *kcpMux) close(ctx context.Context, sock *KCPSocket) {
	mux.closeOnce.Do(func() {
		// 未开启内置协议
		if !mux.isInline {
			return
		}
		if atomic.CompareAndSwapInt32(&mux.state, kmsEstablished, kmsFinWait1) {
			mux.sendInline(ctx, sock, kmsFinWait1)
		}
		ticker := time.NewTicker(kcpCloseTimeout)
		defer ticker.Stop()
		select {
		case <-ticker.C:
			xlog.Get(ctx).Warn("Mux close timeout", zap.Any("state", kmsString(atomic.LoadInt32(&mux.state))), zap.Any("isServer", mux.isListen))
			return
		case <-mux.closeCh:
		}
		xlog.Get(ctx).Info("Mux close", zap.Any("state", kmsString(atomic.LoadInt32(&mux.state))), zap.Any("isServer", mux.isListen))
	})

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
