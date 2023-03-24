package xnet

import (
	"context"
	"fmt"
	"gotu/pkg/xcommon"
	"gotu/pkg/xlog"
	"net"
	"sync"
	"time"

	"github.com/pkg/errors"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type WebsocketArgs struct {
	conn         *websocket.Conn
	onMsg        OnHandlerOnce
	onConnect    OnConnect
	onDisconnect OnDisconnect
}

type Websocket struct {
	conn         *websocket.Conn
	onMsg        OnHandlerOnce
	onConnect    OnConnect
	onDisconnect OnDisconnect

	writeCh chan []byte // 写channel

	closeOnce sync.Once
	closeCh   chan struct{}
	wg        xcommon.WaitGroup
}

func NewWebsocket(ctx context.Context, arg WebsocketArgs) (*Websocket, error) {
	sock := &Websocket{conn: arg.conn,
		onMsg:        arg.onMsg,
		onConnect:    arg.onConnect,
		onDisconnect: arg.onDisconnect,
		writeCh:      make(chan []byte, writeChanLimit),
		closeCh:      make(chan struct{}),
	}
	sock.conn.SetReadLimit(maxMessageSize)

	sock.wg.Add(2)
	go sock.readLoop(ctx)
	go sock.writeLoop(ctx)
	return sock, nil
}

func (sock *Websocket) readLoop(ctx context.Context) {
	var readErr error
	defer func() {
		if readErr != nil {
			xlog.Get(ctx).Warn("Read loop exit with error.", zap.Any("err", readErr))
		}
		sock.forceClose()
	}()

	defer sock.wg.Done(ctx)

	state := sock.onConnect(ctx, sock)
	defer func() {
		sock.onDisconnect(ctx, state)
	}()

	for {
		if err := sock.conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
			readErr = err
			break
		}

		_, message, err := sock.conn.ReadMessage()
		if err != nil {
			if e, ok := err.(*websocket.CloseError); (!ok || e.Code != websocket.CloseNormalClosure) && !errors.Is(err, net.ErrClosed) {
				readErr = err
			}
			break
		}
		// websocket 自动解包, 无需流式处理
		_, err = sock.onMsg(ctx, state, message)
		if err != nil {
			readErr = err
			break
		}
	}
}

func (sock *Websocket) writeLoop(ctx context.Context) {
	var writeErr error
	defer func() {
		if writeErr != nil {
			xlog.Get(ctx).Warn("Write loop exit with error.", zap.Any("err", writeErr))
		}

		if err := sock.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil && err != websocket.ErrCloseSent {
			xlog.Get(ctx).Warn("Write close message failed.", zap.Any("err", err))
		}

		_ = sock.conn.Close()
	}()

	defer sock.wg.Done(ctx)

	close := false

loop:
	for {
		var msg []byte
		if !close {
			// 阻塞获取数据
			select {
			case msg = <-sock.writeCh:
			case <-sock.closeCh:
				close = true
				continue loop
			}
		} else {
			// closed状态,非阻塞获取数据,将待发送数据全部发送
			select {
			case msg = <-sock.writeCh:
			default:
			}
		}
		if msg == nil {
			break
		}

		if err := sock.conn.SetWriteDeadline(time.Now().Add(readTimeout)); err != nil {
			writeErr = err
			break
		}

		if err := sock.conn.WriteMessage(websocket.BinaryMessage, msg); err != nil {
			writeErr = err
			break
		}
	}
}

func (sock *Websocket) Close(ctx context.Context) {
	sock.forceClose()
	sock.wg.Wait()
}

func (sock *Websocket) forceClose() {
	sock.closeOnce.Do(func() {
		close(sock.closeCh)
	})
}

func (sock *Websocket) WaitUntilClose(ctx context.Context) {
	sock.wg.Wait()
}

func (sock *Websocket) SendMsg(ctx context.Context, msg []byte) error {
	select {
	case sock.writeCh <- msg:
		return nil
	case <-sock.closeCh:
		return fmt.Errorf("sock already close")
	default:
		return fmt.Errorf("msg overflow")
	}
}

func (sock *Websocket) RemoteAddr() net.Addr {
	return sock.conn.RemoteAddr()
}
