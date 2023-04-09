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
	arg  *WSCliArgs
}

func NewWSClient(ctx context.Context, arg WSCliArgs) (*WSClient, error) {
	cli := &WSClient{arg: &arg}
	if err := cli.newSocket(ctx); err != nil {
		return nil, err
	}
	return cli, nil
}

func (cli *WSClient) newSocket(ctx context.Context) error {
	u := url.URL{Scheme: "ws", Host: cli.arg.Addr, Path: cli.arg.Path}
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), http.Header{})
	if err != nil {
		return err
	}
	sock, err := NewWebsocket(ctx, WebsocketArgs{
		conn:         conn,
		onMsg:        cli.arg.OnMsg,
		onConnect:    cli.arg.OnConnect,
		onDisconnect: cli.arg.OnDisconnect,
	})
	cli.sock = sock
	if err != nil {
		return err
	}
	return err
}

func (cli *WSClient) Reconnect(ctx context.Context) error {
	cli.sock.Close(ctx)
	return cli.newSocket(ctx)
}

func (cli *WSClient) Close(ctx context.Context) {
	cli.sock.Close(ctx)
}

func (cli *WSClient) SendMsg(ctx context.Context, msg []byte) error {
	return cli.sock.SendMsg(ctx, msg)
}
