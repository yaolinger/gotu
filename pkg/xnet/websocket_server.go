package xnet

import (
	"context"
	"gonet/pkg/xcommon"
	"gonet/pkg/xlog"
	"net"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type WSSvrArgs struct {
	Addr         string
	Path         string
	OnMsg        OnHandlerOnce
	OnConnect    OnConnect
	OnDisconnect OnDisconnect
}

type WSServer struct {
	upgrader *websocket.Upgrader
	httpSrv  *http.Server
	wg       xcommon.WaitGroup

	mu      sync.Mutex
	sockets map[*Websocket]bool // 所有的active连接
}

func NewWSServer(ctx context.Context, arg WSSvrArgs) *WSServer {
	svr := &WSServer{
		upgrader: &websocket.Upgrader{},
		sockets:  make(map[*Websocket]bool),
	}
	// 注册websocket路由
	mux := http.NewServeMux()
	mux.Handle(arg.Path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		conn, err := svr.upgrader.Upgrade(w, r, nil)
		if err != nil {
			xlog.Get(ctx).Error("Upgrade connection failed", zap.Any("err", err))
			// upgrader will respond
			return
		}
		defer conn.Close()

		sock, err := NewWebsocket(ctx, WebsocketArgs{
			conn:         conn,
			onMsg:        arg.OnMsg,
			onConnect:    arg.OnConnect,
			onDisconnect: arg.OnDisconnect,
		})
		if err != nil {
			return
		}
		svr.addSocket(sock)

		sock.WaitUntilClose(ctx)

		svr.delSocket(sock)
	}))

	svr.httpSrv = &http.Server{
		Addr:    arg.Addr,
		Handler: mux,
		BaseContext: func(net.Listener) context.Context {
			// 把传入的context作为每个request的基础context
			return ctx
		},
	}

	svr.wg.Add(1)
	svr.start(ctx)
	return svr
}

func (svr *WSServer) start(ctx context.Context) {
	defer svr.wg.Done(ctx)
	go func() {
		if err := svr.httpSrv.ListenAndServe(); err != nil {
			panic(err)
		}
	}()
}

func (svr *WSServer) Close(ctx context.Context) {
	svr.mu.Lock()
	for sock := range svr.sockets {
		sock.Close(ctx)
	}
	svr.mu.Unlock()
	_ = svr.httpSrv.Close()
	svr.wg.Wait()
}

func (svr *WSServer) addSocket(sock *Websocket) {
	svr.mu.Lock()
	defer svr.mu.Unlock()
	svr.sockets[sock] = true
}

func (svr *WSServer) delSocket(sock *Websocket) {
	svr.mu.Lock()
	defer svr.mu.Unlock()
	delete(svr.sockets, sock)
}
