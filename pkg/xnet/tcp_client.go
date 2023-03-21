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
}

func NewTCPClient(ctx context.Context, arg TCPCliArgs) (*TCPClient, error) {
	tcpAddr, err := net.ResolveTCPAddr(tcpNetwork, arg.Addr)
	if err != nil {
		return nil, err
	}

	conn, err := net.DialTCP(tcpNetwork, nil, tcpAddr)
	if err != nil {
		return nil, err
	}

	bufMgr := newBufferManager()

	sock := newTCPSocket(ctx, TCPSocketArgs{
		conn:           conn,
		readBufferPool: bufMgr.newBufferPool(),
		onMsg:          arg.OnMsg,
		onConnect:      arg.OnConnect,
		onDisconnect:   arg.OnDisconnect,
		releaseFn:      func(ctx context.Context, ts *TCPSocket) {},
	})
	return &TCPClient{sock: sock, bufMgr: bufMgr}, nil
}

func (cli *TCPClient) Close(ctx context.Context) {
	cli.sock.Close(ctx)
}

func (cli *TCPClient) SendMsg(ctx context.Context, msg []byte) error {
	return cli.sock.SendMsg(ctx, msg)
}
