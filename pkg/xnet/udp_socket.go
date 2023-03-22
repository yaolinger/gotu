package xnet

import (
	"context"
	"errors"
	"fmt"
	"gonet/pkg/xcommon"
	"gonet/pkg/xlog"
	"net"
	"sync"

	"go.uber.org/zap"
)

type UDPSocketArgs struct {
	isServer bool
	conn     *net.UDPConn
	onMsg    udpOnMsg
}

type udpDatagram struct {
	msg  []byte
	addr *net.UDPAddr
}

type UDPSocket struct {
	isServer bool
	conn     *net.UDPConn
	onMsg    udpOnMsg
	writeCh  chan *udpDatagram

	closeOnce sync.Once
	closeCh   chan struct{}
	wg        xcommon.WaitGroup
}

func NewUDPSocket(ctx context.Context, arg UDPSocketArgs) *UDPSocket {
	sock := &UDPSocket{
		isServer: arg.isServer,
		conn:     arg.conn,
		onMsg:    arg.onMsg,
		writeCh:  make(chan *udpDatagram, writeChanLimit*10),
		closeCh:  make(chan struct{}),
	}
	sock.wg.Add(2)
	go sock.readLoop(ctx)
	go sock.writeLoop(ctx)
	return sock
}

func (sock *UDPSocket) readLoop(ctx context.Context) {
	var readErr error
	defer func() {
		if readErr != nil {
			xlog.Get(ctx).Warn("Read loop exit with error", zap.Any("err", readErr))
		}
		sock.forceClose(ctx)
	}()

	defer sock.wg.Done(ctx)
	for {
		bytes := make([]byte, readBufferSize)

		// TODO 错误分析, 是否出错即关闭
		n, addr, err := sock.conn.ReadFromUDP(bytes)
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				readErr = err
			}
			break
		}
		sock.onMsg(ctx, bytes[0:n], addr)
	}
}

func (sock *UDPSocket) writeLoop(ctx context.Context) {
	var writeErr error
	defer func() {
		if writeErr != nil {
			xlog.Get(ctx).Warn("Write loop exit with error", zap.Any("err", writeErr))
		}

		_ = sock.conn.Close()
	}()

	defer sock.wg.Done(ctx)

	close := false

loop:
	for {
		var datagram *udpDatagram
		if !close {
			select {
			case datagram = <-sock.writeCh:
			case <-sock.closeCh:
				close = true
				continue loop
			}
		} else {
			select {
			case datagram = <-sock.writeCh:
			default:
			}
		}
		if datagram == nil {
			break
		}

		// TODO 错误分析, 是否出错即关闭
		if sock.isServer {
			_, err := sock.conn.WriteToUDP(datagram.msg, datagram.addr)
			if err != nil {
				writeErr = err
				break
			}
		} else {
			_, err := sock.conn.Write(datagram.msg)
			if err != nil {
				writeErr = err
				break
			}
		}
	}
}

func (sock *UDPSocket) close(ctx context.Context) {
	sock.forceClose(ctx)
	sock.wg.Wait()
}

func (sock *UDPSocket) forceClose(ctx context.Context) {
	sock.closeOnce.Do(func() {
		close(sock.closeCh)
	})
}

func (sock *UDPSocket) sendMsg(ctx context.Context, datagram *udpDatagram) error {
	select {
	case sock.writeCh <- datagram:
		return nil
	case <-sock.closeCh:
		return fmt.Errorf("sock already close")
	default:
		return fmt.Errorf("msg overflow")
	}
}
