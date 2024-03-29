package xnet

import (
	"context"
	"sync"

	"gotu/pkg/xlog"

	"github.com/xtaci/kcp-go"
	"go.uber.org/zap"
)

type KCPServerArgs struct {
	Addr         string
	OnMsg        OnHandlerOnce
	OnConnect    OnConnect
	OnDisconnect OnDisconnect
	IsInline     bool // 是否开启内置协议(握手，挥手)
}

type KCPServer struct {
	wg       sync.WaitGroup
	listener *kcp.Listener // 监听器
	closeCh  chan struct{}

	bufMgr       *bufferManager
	onMsg        OnHandlerOnce
	onConnect    OnConnect
	onDisconnect OnDisconnect
	isInline     bool

	mu      sync.Mutex
	sockets map[*KCPSocket]bool
}

func NewKCPServer(ctx context.Context, arg KCPServerArgs) (*KCPServer, error) {
	listener, err := kcp.ListenWithOptions(arg.Addr, nil, kcpFecDataShards, kcpFecParityShards)
	if err != nil {
		return nil, err
	}
	xlog.Get(ctx).Info("KCP server start listen success.", zap.String("addr", arg.Addr))

	svr := &KCPServer{
		listener:     listener,
		closeCh:      make(chan struct{}),
		sockets:      map[*KCPSocket]bool{},
		bufMgr:       newBufferManager(),
		onMsg:        arg.OnMsg,
		onConnect:    arg.OnConnect,
		onDisconnect: arg.OnDisconnect,
		isInline:     arg.IsInline,
	}

	svr.wg.Add(1)
	go svr.accept(ctx)
	return svr, nil
}

func (svr *KCPServer) accept(ctx context.Context) {
	defer svr.wg.Done()
	for {
		conn, err := svr.listener.AcceptKCP()

		// 监听关闭检测
		select {
		case <-svr.closeCh:
			xlog.Get(ctx).Debug("KCP listener close.")
			return
		default:
		}

		if err != nil {
			xlog.Get(ctx).Warn("Accept kcp failed.", zap.Any("err", err))
			continue
		}
		ks, err := newKCPSocket(ctx, kcpSocketArgs{
			conn:         conn,
			readBufPool:  svr.bufMgr.newBufferPool(),
			mux:          newKCPMux(svr.onMsg, svr.isInline, true),
			onConnect:    svr.onConnect,
			onDisconnect: svr.onDisconnect,
			releaseFn:    svr.deleteSocket,
		})
		if err != nil {
			xlog.Get(ctx).Warn("New kcp socket failed.", zap.Any("err", err))
			continue
		}
		svr.addSocket(ctx, ks)
	}
}

func (svr *KCPServer) Close(ctx context.Context) {
	svr.mu.Lock()
	defer svr.mu.Unlock()
	for xs := range svr.sockets {
		xs.Close(ctx)
	}

	close(svr.closeCh)
	svr.listener.Close()
	svr.wg.Wait()
	xlog.Get(ctx).Debug("KCP server close success.")
}

func (svr *KCPServer) addSocket(ctx context.Context, sock *KCPSocket) {
	svr.mu.Lock()
	defer svr.mu.Unlock()
	svr.sockets[sock] = true

	xlog.Get(ctx).Debug("Add sockets", zap.Any("count", len(svr.sockets)))
}

func (svr *KCPServer) deleteSocket(ctx context.Context, sock *KCPSocket) {
	sock.Close(ctx)

	svr.mu.Lock()
	defer svr.mu.Unlock()
	delete(svr.sockets, sock)

	xlog.Get(ctx).Debug("Del sockets", zap.Any("count", len(svr.sockets)))
}
