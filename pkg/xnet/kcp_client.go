package xnet

import (
	"context"

	"github.com/xtaci/kcp-go"
)

type KCPClient struct {
	sock   *KCPSocket
	bufMgr *bufferManager
}

type KCPClientArgs struct {
	Addr         string
	OnMsg        OnHandlerOnce
	OnConnect    OnConnect
	OnDisconnect OnDisconnect

	// Fin          PackKCPFinalMsg
}

func NewKCPClient(ctx context.Context, arg KCPClientArgs) (*KCPClient, error) {
	conn, err := kcp.DialWithOptions(arg.Addr, nil, 0, 0)
	if err != nil {
		return nil, err
	}
	bufMgr := newBufferManager()

	sock := newKCPSocket(ctx, kcpSocketArgs{
		conn:         conn,
		onMsg:        arg.OnMsg,
		onConnect:    arg.OnConnect,
		onDisconnect: arg.OnDisconnect,
		releaseFn:    func(ctx context.Context, sock *KCPSocket) {},
		readBufPool:  bufMgr.newBufferPool(),
		//fin:          arg.Fin,
	})
	return &KCPClient{sock: sock, bufMgr: bufMgr}, nil
}

func (cli *KCPClient) Close(ctx context.Context) {
	cli.sock.Close(ctx)
}

func (cli *KCPClient) SendMsg(ctx context.Context, msg []byte) error {
	return cli.sock.SendMsg(ctx, msg)
}
