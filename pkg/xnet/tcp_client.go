package xnet

import (
	"context"
	"net"
)

type TCPCliArgs struct {
	Addr         string
	OnMsg        OnHandlerOnce
	OnConnect    OnConnect
	OnDisconnect OnDisconnect
}

type TCPClient struct {
	sock   *TCPSocket
	bufMgr *bufferManager
	arg    *TCPCliArgs
}

func NewTCPClient(ctx context.Context, arg TCPCliArgs) (*TCPClient, error) {
	bufMgr := newBufferManager()

	cli := &TCPClient{bufMgr: bufMgr, arg: &arg}
	if err := cli.newSocket(ctx); err != nil {
		return nil, err
	}
	return cli, nil
}

func (cli *TCPClient) newSocket(ctx context.Context) error {
	tcpAddr, err := net.ResolveTCPAddr(tcpNetwork, cli.arg.Addr)
	if err != nil {
		return err
	}
	conn, err := net.DialTCP(tcpNetwork, nil, tcpAddr)
	if err != nil {
		return err
	}
	cli.sock = newTCPSocket(ctx, TCPSocketArgs{
		conn:           conn,
		readBufferPool: cli.bufMgr.newBufferPool(),
		onMsg:          cli.arg.OnMsg,
		onConnect:      cli.arg.OnConnect,
		onDisconnect:   cli.arg.OnDisconnect,
		releaseFn:      func(ctx context.Context, ts *TCPSocket) {},
	})
	return nil
}

func (cli *TCPClient) Reconnect(ctx context.Context) error {
	cli.sock.Close(ctx)
	return cli.newSocket(ctx)
}

func (cli *TCPClient) Close(ctx context.Context) {
	cli.sock.Close(ctx)
}

func (cli *TCPClient) SendMsg(ctx context.Context, msg []byte) error {
	return cli.sock.SendMsg(ctx, msg)
}
