package xnet

import (
	"context"
	"gonet/pkg/xcommon"
	"gonet/pkg/xlog"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

type UDPSvrArgs struct {
	Addr         string
	OnMsg        OnHandlerOnce
	OnConnect    OnConnect
	OnDisconnect OnDisconnect
}

type UDPServer struct {
	onMsg        OnHandlerOnce
	onConnect    OnConnect
	onDisconnect OnDisconnect

	sock *UDPSocket

	mu       sync.Mutex
	sessions map[string]*UDPSession

	closeCh chan struct{}
	wg      xcommon.WaitGroup
}

func NewUDPServer(ctx context.Context, arg UDPSvrArgs) (*UDPServer, error) {
	udpAddr, err := net.ResolveUDPAddr(udpNetwork, arg.Addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP(udpNetwork, udpAddr)
	if err != nil {
		return nil, err
	}
	svr := &UDPServer{
		onMsg:        arg.OnMsg,
		onConnect:    arg.OnConnect,
		onDisconnect: arg.OnDisconnect,
		sessions:     make(map[string]*UDPSession),
		closeCh:      make(chan struct{}),
	}

	sock := NewUDPSocket(ctx, UDPSocketArgs{isServer: true, conn: conn, onMsg: svr.udpOnMsg})
	svr.sock = sock

	svr.wg.Add(1)
	go svr.checkLoop(ctx)
	return svr, nil
}

func (svr *UDPServer) checkLoop(ctx context.Context) {
	defer svr.wg.Done(ctx)

	ticker := time.NewTicker(udpCheckDuration)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-ticker.C:
		case <-svr.closeCh:
			break loop
		}
		sessionTimeout := time.Now().Unix() - udpSessionTimeout
		expires := make([]*UDPSession, 0)

		svr.mu.Lock()
		for _, session := range svr.sessions {
			if session.getActiveAt() < sessionTimeout {
				expires = append(expires, session)
			}
		}
		svr.mu.Unlock()

		for _, session := range expires {
			svr.delSession(ctx, session)
			xlog.Get(ctx).Warn("UDP session timeout", zap.Any("id", addrToString(session.remoteDddr())))
		}
	}
}

func (svr *UDPServer) udpOnMsg(ctx context.Context, msg []byte, addr *net.UDPAddr) {
	id := addrToString(addr)
	now := time.Now().Unix()
	session := svr.getSession(ctx, id)
	if session == nil {
		subCtx, cancel := context.WithCancel(ctx)
		session = NewUDPSession(subCtx, UDPSessionArgs{
			cancel:       cancel,
			addr:         addr,
			onMsg:        svr.onMsg,
			onConnect:    svr.onConnect,
			onDisconnect: svr.onDisconnect,
			sendMsg:      svr.sock.sendMsg,
			now:          now,
		})
		svr.addSession(ctx, session)
	}
	if err := session.recvMsg(msg, now); err != nil {
		xlog.Get(ctx).Warn("UDP session  recv msg failed.", zap.Any("err", err))
	}
}

func (svr *UDPServer) addSession(ctx context.Context, session *UDPSession) {
	id := addrToString(session.remoteDddr())
	svr.mu.Lock()
	defer svr.mu.Unlock()
	svr.sessions[id] = session
}

func (svr *UDPServer) delSession(ctx context.Context, session *UDPSession) {
	id := addrToString(session.remoteDddr())
	svr.mu.Lock()
	defer svr.mu.Unlock()
	delete(svr.sessions, id)
}

func (svr *UDPServer) getSession(ctx context.Context, id string) *UDPSession {
	svr.mu.Lock()
	defer svr.mu.Unlock()
	return svr.sessions[id]
}

func (svr *UDPServer) Close(ctx context.Context) {
	svr.mu.Lock()
	for _, session := range svr.sessions {
		session.Close(ctx)
	}
	svr.mu.Unlock()

	svr.sock.close(ctx)
	close(svr.closeCh)
	svr.wg.Wait()
}
