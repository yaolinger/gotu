package xnet

import (
	"context"

	"github.com/xtaci/kcp-go"
)

type KCPClient struct {
	sock   *KCPSocket
	bufMgr *bufferManager

	arg *KCPClientArgs
}

type KCPClientArgs struct {
	Addr         string
	OnMsg        OnHandlerOnce
	OnConnect    OnConnect
	OnDisconnect OnDisconnect
	IsInline     bool // 是否开启内置协议(握手，挥手)
}

func NewKCPClient(ctx context.Context, arg KCPClientArgs) (*KCPClient, error) {
	bufMgr := newBufferManager()
	cli := &KCPClient{bufMgr: bufMgr, arg: &arg}
	if err := cli.newSocket(ctx); err != nil {
		return nil, err
	}
	return cli, nil
}

func (cli *KCPClient) newSocket(ctx context.Context) error {
	conn, err := kcp.DialWithOptions(cli.arg.Addr, nil, kcpFecDataShards, kcpFecParityShards)
	if err != nil {
		return err
	}
	sock, err := newKCPSocket(ctx, kcpSocketArgs{
		conn:         conn,
		mux:          newKCPMux(cli.arg.OnMsg, cli.arg.IsInline, false),
		onConnect:    cli.arg.OnConnect,
		onDisconnect: cli.arg.OnDisconnect,
		releaseFn:    func(ctx context.Context, sock *KCPSocket) {},
		readBufPool:  cli.bufMgr.newBufferPool(),
	})
	cli.sock = sock
	return err
}

func (cli *KCPClient) Reconnect(ctx context.Context) error {
	cli.sock.Close(ctx)
	return cli.newSocket(ctx)
}

func (cli *KCPClient) Close(ctx context.Context) {
	cli.sock.Close(ctx)
}

func (cli *KCPClient) SendMsg(ctx context.Context, msg []byte) error {
	return cli.sock.SendMsg(ctx, msg)
}
