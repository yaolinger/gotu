package xnet

import (
	"context"
	"fmt"
	"gonet/pkg/xcommon"
	"gonet/pkg/xlog"
	"net"
	"sync/atomic"

	"sync"

	"go.uber.org/zap"
)

type UDPSessionArgs struct {
	cancel       context.CancelFunc
	addr         *net.UDPAddr
	onMsg        OnHandlerOnce
	onConnect    OnConnect
	onDisconnect OnDisconnect
	sendMsg      udpSendMsg
	now          int64
}

type UDPSession struct {
	cancel       context.CancelFunc
	addr         *net.UDPAddr
	onMsg        OnHandlerOnce
	onConnect    OnConnect
	onDisconnect OnDisconnect
	sendMsg      udpSendMsg
	activeAt     int64

	msgCh chan []byte

	closeCh   chan struct{}
	closeOnce sync.Once
	wg        xcommon.WaitGroup
}

func NewUDPSession(ctx context.Context, arg UDPSessionArgs) *UDPSession {
	session := &UDPSession{
		cancel:       arg.cancel,
		addr:         arg.addr,
		onMsg:        arg.onMsg,
		onConnect:    arg.onConnect,
		onDisconnect: arg.onDisconnect,
		sendMsg:      arg.sendMsg,
		activeAt:     arg.now,
		msgCh:        make(chan []byte, udpMsgChanLimit),
		closeCh:      make(chan struct{}),
	}
	session.wg.Add(1)
	go session.handlerLoop(ctx)
	return session
}

func (session *UDPSession) handlerLoop(ctx context.Context) {
	var handlerErr error
	defer func() {
		if handlerErr != nil {
			xlog.Get(ctx).Warn("Handler loop exit with error", zap.Any("err", handlerErr))
		}
	}()

	defer session.wg.Done(ctx)

	state := session.onConnect(ctx, session)
	defer func() {
		session.onDisconnect(ctx, session)
	}()

loop:
	for {
		var msg []byte
		select {
		case msg = <-session.msgCh:
		case <-session.closeCh:
			break loop
		}

		_, err := session.onMsg(ctx, state, msg)
		if err != nil {
			handlerErr = err
			break
		}
	}
}

func (session *UDPSession) recvMsg(msg []byte, now int64) error {
	select {
	case session.msgCh <- msg:
		atomic.StoreInt64(&session.activeAt, now)
		return nil
	case <-session.closeCh:
		return fmt.Errorf("session already close")
	default:
		return fmt.Errorf("session overflow")
	}
}

func (session *UDPSession) remoteDddr() *net.UDPAddr {
	return session.addr
}

func (session *UDPSession) getActiveAt() int64 {
	return atomic.LoadInt64(&session.activeAt)
}

func (session *UDPSession) Close(ctx context.Context) {
	session.forceClose(ctx)
	session.wg.Wait()
}

func (session *UDPSession) forceClose(ctx context.Context) {
	session.closeOnce.Do(func() {
		close(session.closeCh)
	})
}

func (session *UDPSession) SendMsg(ctx context.Context, msg []byte) error {
	return session.sendMsg(ctx, &udpDatagram{msg: msg, addr: session.addr})
}
