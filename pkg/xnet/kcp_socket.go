package xnet

import (
	"context"
	"fmt"
	"gotu/pkg/xcommon"
	"gotu/pkg/xlog"
	"io"
	"net"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"github.com/xtaci/kcp-go"
	"go.uber.org/zap"
)

type kcpSocketArgs struct {
	conn         *kcp.UDPSession
	onConnect    OnConnect
	onDisconnect OnDisconnect
	mux          *kcpMux
	releaseFn    func(ctx context.Context, sock *KCPSocket)

	readBufPool *bufferPool
}

type KCPSocket struct {
	conn         *kcp.UDPSession
	readBufPool  *bufferPool
	onConnect    OnConnect
	onDisconnect OnDisconnect
	releaseFn    func(ctx context.Context, sock *KCPSocket)
	readCaches   []byte
	writeCh      chan []byte // 写消息缓存
	mux          *kcpMux

	wg xcommon.WaitGroup

	closeFlag int32         // 关闭标识
	closeCh   chan struct{} // 关闭Channel
}

func newKCPSocket(ctx context.Context, arg kcpSocketArgs) (*KCPSocket, error) {
	arg.conn.SetStreamMode(true)
	arg.conn.SetWriteDelay(false)
	arg.conn.SetACKNoDelay(kcpAckNoDelay)
	arg.conn.SetNoDelay(kcpNoDelay, kcpInterval, kcpResend, kcpNC)

	sock := &KCPSocket{
		conn:         arg.conn,
		readBufPool:  arg.readBufPool,
		onConnect:    arg.onConnect,
		onDisconnect: arg.onDisconnect,
		releaseFn:    arg.releaseFn,
		readCaches:   make([]byte, 0),
		writeCh:      make(chan []byte, writeChanLimit),
		mux:          arg.mux,
		closeCh:      make(chan struct{}),
		closeFlag:    kcpSocketStart,
	}

	sock.wg.Add(2)
	go sock.readLoop(ctx)
	go sock.writeLoop(ctx)

	if err := sock.mux.init(ctx, sock); err != nil {
		sock.closeForce(ctx)
		return nil, err
	}
	return sock, nil
}

func (sock *KCPSocket) readLoop(ctx context.Context) {
	state := sock.onConnect(ctx, sock)

	var readErr error
	defer func() {
		if readErr != nil {
			xlog.Get(ctx).Warn("Read loop exit with error.", zap.Any("err", readErr))
		}
		sock.onDisconnect(ctx, state)
		sock.releaseFn(ctx, sock)
		sock.closeOnce()
	}()

	defer sock.wg.Done(ctx)

	for {
		// timeout
		if err := sock.conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
			readErr = err
			break
		}

		// 分配buf, 读取数据
		bytes := sock.readBufPool.get()
		n, err := sock.conn.Read(bytes)
		if atomic.LoadInt32(&sock.closeFlag) == kcpSocketClose || err != nil {
			if err != nil && errors.Cause(err) != io.ErrClosedPipe && !errors.Is(err, net.ErrClosed) {
				readErr = err
			}
			break
		}

		sock.readBufPool.put(bytes[n:])
		sock.readCaches = append(sock.readCaches, bytes[0:n]...)

		isKeepCache := false
		for !isKeepCache {
			reqCount, err := sock.mux.onMsg(ctx, sock, state, sock.readCaches)
			if err != nil {
				if err != io.EOF {
					readErr = err
				}
				return
			}
			if reqCount == 0 {
				// 缓存数据无法处理(长度不够)
				isKeepCache = true
			} else {
				sock.readBufPool.put(sock.readCaches[0:reqCount])
				sock.readCaches = sock.readCaches[reqCount:]
			}
		}
	}
}

func (sock *KCPSocket) write(msg []byte) error {
	for {
		if err := sock.conn.SetWriteDeadline(time.Now().Add(writeTimeout)); err != nil {
			return err
		}
		n, err := sock.conn.Write(msg)
		if err != nil {
			return err
		}
		if n >= len(msg) {
			break
		}
		msg = msg[n:]
	}
	return nil
}

func (sock *KCPSocket) writeLoop(ctx context.Context) {
	var writeErr error
	defer func() {
		if writeErr != nil {
			xlog.Get(ctx).Warn("Write loop exit with error", zap.Any("err", writeErr))
		}
		// 写入全部数据后关闭 udpsession
		_ = sock.conn.Close()
	}()

	defer sock.wg.Done(ctx)

	waitMsg := func() ([]byte, bool) {
		// 阻塞等待数据
		var msg []byte
		ret := false
		select {
		case data := <-sock.writeCh:
			msg = append(msg, data...)
		case <-sock.closeCh:
			ret = true
		}
		// 非阻塞获取数据
	loop:
		for {
			select {
			case data := <-sock.writeCh:
				msg = append(msg, data...)
			default:
				break loop
			}
		}
		return msg, ret
	}

	isClosed := false
	for !isClosed {
		var msg []byte
		msg, isClosed = waitMsg()
		if err := sock.write(msg); err != nil {
			writeErr = err
			break
		}
	}
}

// 逻辑层调用
func (sock *KCPSocket) SendMsg(ctx context.Context, payload []byte) error {
	return sock.send(ctx, false, payload)
}

// 内置发送
func (sock *KCPSocket) send(ctx context.Context, inline bool, payload []byte) error {
	msg, err := sock.mux.packMsg(inline, payload)
	if err != nil {
		return err
	}
	select {
	case sock.writeCh <- msg:
		return nil
	case <-sock.closeCh:
		return fmt.Errorf("sock already close")
	default:
		return fmt.Errorf("msg overflow")
	}
}

func (sock *KCPSocket) Close(ctx context.Context) {
	sock.mux.close(ctx, sock)
	sock.closeForce(ctx)
}

func (sock *KCPSocket) closeForce(ctx context.Context) {
	sock.closeOnce()
	sock.wg.Wait()
}

// kcp.udpSession 关闭
// 因为conn.Close()无法控制读写关闭时序, 采用atomic, close channel 控制关闭流程
// close(closeCh) => write loop => read loop => conn.Close()
func (sock *KCPSocket) closeOnce() {
	if atomic.CompareAndSwapInt32(&sock.closeFlag, kcpSocketStart, kcpSocketClose) {
		close(sock.closeCh)
	}
}

func (sock *KCPSocket) RemoteAddr() net.Addr {
	return sock.conn.RemoteAddr()
}
