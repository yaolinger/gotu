package xnet

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"gonet/pkg/xlog"
	"io"
	"sync/atomic"

	"go.uber.org/zap"
)

var (
	kcpMuxHeaderSizeof       = binary.Size(&kcpMuxHeader{})
	kcpExchangeSizeof        = binary.Size(&kcpExchange{})
	kmsEstablished     int32 = 1
	kmsClosed          int32 = 2
)

type kcpMuxHeader struct {
	Inline bool // 是否为内置协议
}

// kcp 内置协议
type kcpExchange struct {
	State int32 // 状态
}

type kcpMux struct {
	state   int32
	handler OnHandlerOnce
}

func newKCPMux(handler OnHandlerOnce) *kcpMux {
	m := &kcpMux{state: kmsEstablished, handler: handler}
	return m
}

// 处理消息
func (mux *kcpMux) onMsg(ctx context.Context, state interface{}, msg []byte) (int, error) {
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
			return 0, fmt.Errorf("state [%v] not established", atomic.LoadInt32(&mux.state))
		}
	} else {
		// 内置协议
		ke := &kcpExchange{}
		ioReader := bytes.NewReader(msg[kcpMuxHeaderSizeof : kcpMuxHeaderSizeof+kcpExchangeSizeof])
		if err := binary.Read(ioReader, binary.LittleEndian, ke); err != nil {
			return 0, err
		}

		// TODO(补全握手挥手): 简化处理close状态
		if ke.State == kmsClosed {
			return 0, io.EOF
		}
		return 0, fmt.Errorf("inline invalid")
	}
}

func (mux *kcpMux) close(ctx context.Context, sock *KCPSocket) {
	ke := &kcpExchange{State: kmsClosed}
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

// 补充kcp包头
func (mux *kcpMux) packMsg(inline bool, payload []byte) ([]byte, error) {
	header := &kcpMuxHeader{Inline: inline}
	ioWrite := bytes.NewBuffer(nil)
	if err := binary.Write(ioWrite, binary.LittleEndian, header); err != nil {
		return nil, err
	}
	msg := ioWrite.Bytes()
	msg = append(msg, payload...)
	return msg, nil
}
