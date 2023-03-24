package xnet

import (
	"context"
	"fmt"
	"gotu/pkg/xcommon"
	"gotu/pkg/xlog"
	"io"
	"net"
	"time"

	"go.uber.org/zap"
)

type TCPSocketArgs struct {
	conn           *net.TCPConn
	readBufferPool *bufferPool
	onMsg          OnHandlerOnce
	onConnect      OnConnect
	onDisconnect   OnDisconnect
	releaseFn      func(ctx context.Context, ts *TCPSocket)
}

type TCPSocket struct {
	conn           *net.TCPConn
	readBufferPool *bufferPool
	readCaches     []byte
	writeCh        chan []byte   // 写消息缓存
	closeCh        chan struct{} // 关闭channel

	onMsg        OnHandlerOnce
	onConnect    OnConnect
	onDisconnect OnDisconnect
	releaseFn    func(ctx context.Context, ts *TCPSocket)

	wg xcommon.WaitGroup
}

func newTCPSocket(ctx context.Context, arg TCPSocketArgs) *TCPSocket {
	// arg.conn.SetNoDelay(true)
	// arg.conn.SetReadBuffer
	// arg.conn.SetWriteBuffer

	s := &TCPSocket{
		conn:           arg.conn,
		readBufferPool: arg.readBufferPool,
		readCaches:     make([]byte, 0),
		writeCh:        make(chan []byte, writeChanLimit),
		closeCh:        make(chan struct{}),
		onMsg:          arg.onMsg,
		onConnect:      arg.onConnect,
		onDisconnect:   arg.onDisconnect,
		releaseFn:      arg.releaseFn,
	}

	s.wg.Add(2)
	go s.readLoop(ctx)
	go s.writeLoop(ctx)
	return s
}

func (sock *TCPSocket) readLoop(ctx context.Context) {
	state := sock.onConnect(ctx, sock)

	var readErr error
	defer func() {
		if readErr != nil {
			xlog.Get(ctx).Warn("Read loop exit with error.", zap.Any("err", readErr))
		}
		close(sock.closeCh)
		sock.onDisconnect(ctx, state)
	}()
	defer sock.wg.Done(ctx)

	for {
		if err := sock.conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
			readErr = err
			break
		}

		// 获取buffer, 读取buffer
		bytes := sock.readBufferPool.get()
		n, err := sock.conn.Read(bytes)
		if err != nil {
			if err != io.EOF {
				readErr = err
			}
			break
		}

		// 归还剩余buffer
		sock.readBufferPool.put(bytes[n:])
		// 合并cache
		sock.readCaches = append(sock.readCaches, bytes[0:n]...)
		isKeepCache := false
		for !isKeepCache {
			// do handler
			reqCount, err := sock.onMsg(ctx, state, sock.readCaches)
			if err != nil {
				readErr = err
				return
			}
			if reqCount == 0 {
				isKeepCache = true
			} else {
				// 归还已处理buffer
				sock.readBufferPool.put(sock.readCaches[0:reqCount])
				sock.readCaches = sock.readCaches[reqCount:]
			}
		}
	}
}

func (sock *TCPSocket) writeLoop(ctx context.Context) {
	var writeErr error
	defer func() {
		if writeErr != nil {
			xlog.Get(ctx).Warn("Write loop exit with error", zap.Any("err", writeErr))
		}
		_ = sock.conn.Close()
		sock.releaseFn(ctx, sock)
	}()

	defer sock.wg.Done(ctx)

	waitMsg := func() ([]byte, bool) {
		// 阻塞并等待数据
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
		if len(msg) == 0 {
			break
		}
		if err := sock.write(msg); err != nil {
			writeErr = err
			break
		}
	}
}

// 写数据
func (sock *TCPSocket) write(msg []byte) error {
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

func (sock *TCPSocket) SendMsg(ctx context.Context, msg []byte) error {
	select {
	case sock.writeCh <- msg:
		return nil
	case <-sock.closeCh:
		return fmt.Errorf("sock already close")
	default:
		return fmt.Errorf("msg overflow")
	}
}

// close =》 read loop => closeCh =》write loop
func (sock *TCPSocket) Close(ctx context.Context) {
	sock.conn.CloseRead()
	sock.wg.Wait()
}

func (sock *TCPSocket) RemoteAddr() net.Addr {
	return sock.conn.RemoteAddr()
}
