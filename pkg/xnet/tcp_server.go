package xnet

import (
	"context"
	"fmt"
	"gonet/pkg/xlog"
	"net"
	"sync"

	"go.uber.org/zap"
)

type TCPSvrArgs struct {
	Addr  string
	OnMsg OnHandlerOnce
}

type TCPServer struct {
	wg       sync.WaitGroup
	listener *net.TCPListener
	closeCh  chan struct{}

	onMsg OnHandlerOnce

	bufMgr *bufferManager

	mu      sync.Mutex
	sockets map[*TCPSocket]bool
}

func NewTCPServer(ctx context.Context, arg TCPSvrArgs) (*TCPServer, error) {
	tcpAddr, err := net.ResolveTCPAddr(tcpNetwork, arg.Addr)
	if err != nil {
		return nil, err
	}
	listener, err := net.ListenTCP(tcpNetwork, tcpAddr)
	if err != nil {
		return nil, fmt.Errorf("listen addr[%s] failed %w", arg.Addr, err)
	}
	svr := &TCPServer{
		listener: listener,
		closeCh:  make(chan struct{}),
		sockets:  make(map[*TCPSocket]bool),
		bufMgr:   newBufferManager(),
		onMsg:    arg.OnMsg,
	}
	svr.wg.Add(1)
	go svr.accept(ctx)
	xlog.Get(ctx).Info("Start listen success.", zap.String("addr", arg.Addr))
	return svr, nil
}

func (svr *TCPServer) accept(ctx context.Context) {
	defer svr.wg.Done()

	for {
		conn, err := svr.listener.AcceptTCP()

		// 监听关闭检测
		select {
		case <-svr.closeCh:
			xlog.Get(ctx).Debug("Tcp listener close.")
			return
		default:
		}

		if err != nil {
			xlog.Get(ctx).Warn("Accept tcp failed.")
			continue
		}
		s := newTCPSocket(ctx, TCPSocketArgs{
			conn:           conn,
			readBufferPool: svr.bufMgr.newBufferPool(),
			onMsg:          svr.onMsg,
		})
		svr.addSocket(ctx, s)
	}
}

func (svr *TCPServer) Close(ctx context.Context) {
	close(svr.closeCh)
	svr.listener.Close()
	svr.wg.Wait()

	xlog.Get(ctx).Info("TCP server stop.")
}

func (svr *TCPServer) addSocket(ctx context.Context, s *TCPSocket) {
	svr.mu.Lock()
	defer svr.mu.Unlock()
	svr.sockets[s] = true
}

func (svr *TCPServer) delSocket(ctx context.Context, s *TCPSocket) {
	svr.mu.Lock()
	defer svr.mu.Unlock()

	delete(svr.sockets, s)
}
