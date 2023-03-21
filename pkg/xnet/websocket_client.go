package xnet

import (
	"context"
	"net/http"
	"net/url"

	"github.com/gorilla/websocket"
)

type WSCliArgs struct {
	Addr         string
	Path         string
	OnMsg        OnHandlerOnce
	OnConnect    OnConnect
	OnDisconnect OnDisconnect
}

type WSClient struct {
	sock *Websocket
}

func NewWSClient(ctx context.Context, arg WSCliArgs) (*WSClient, error) {
	u := url.URL{Scheme: "ws", Host: arg.Addr, Path: arg.Path}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), http.Header{})
	if err != nil {
		return nil, err
	}
	sock, err := NewWebsocket(ctx, WebsocketArgs{
		conn:         conn,
		onMsg:        arg.OnMsg,
		onConnect:    arg.OnConnect,
		onDisconnect: arg.OnDisconnect,
	})
	if err != nil {
		return nil, err
	}
	return &WSClient{sock: sock}, nil
}

func (cli *WSClient) Close(ctx context.Context) {
	cli.sock.Close(ctx)
}

func (cli *WSClient) SendMsg(ctx context.Context, msg []byte) error {
	return cli.sock.SendMsg(ctx, msg)
}
