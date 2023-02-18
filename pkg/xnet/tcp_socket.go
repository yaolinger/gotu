package xnet

import (
	"context"
	"gonet/pkg/xlog"
	"io"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

type TCPSocketArgs struct {
	conn           *net.TCPConn
	readBufferPool *bufferPool
	onMsg          OnHandlerOnce
}

type TCPSocket struct {
	conn           *net.TCPConn
	readBufferPool *bufferPool
	readCaches     []byte

	onMsg OnHandlerOnce

	wg sync.WaitGroup
}

// TODO
func newTCPSocket(ctx context.Context, arg TCPSocketArgs) *TCPSocket {
	// arg.conn.SetNoDelay(true)
	// arg.conn.SetReadBuffer
	// arg.conn.SetWriteBuffer

	s := &TCPSocket{
		conn:           arg.conn,
		readBufferPool: arg.readBufferPool,
		readCaches:     make([]byte, 0),
		onMsg:          arg.onMsg,
	}

	s.wg.Add(2)
	go s.readLoop(ctx)
	go s.writeLoop(ctx)
	return s
}

func (sock *TCPSocket) readLoop(ctx context.Context) {
	var readErr error

	defer func() {
		if readErr != nil {
			xlog.Get(ctx).Warn("Read loop exit with error.", zap.Any("err", readErr))
		}
	}()

	defer sock.wg.Done()

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
			reqCount, err := sock.onMsg(ctx, sock.readCaches)
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
	defer sock.wg.Done()
	// TODO
}

func (sock *TCPSocket) Close(ctx context.Context) {
	//close(sock.closeCh)
	sock.conn.CloseRead()
	sock.wg.Wait()
}
