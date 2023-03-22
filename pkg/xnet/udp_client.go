package xnet

import (
	"context"
	"net"
	"time"
)

type UDPCliArgs struct {
	Addr         string
	OnMsg        OnHandlerOnce
	OnConnect    OnConnect
	OnDisconnect OnDisconnect
}

// TODO 暂未实现超时机制
type UDPClient struct {
	sock    *UDPSocket
	session *UDPSession
}

func NewUDPClient(ctx context.Context, arg UDPCliArgs) (*UDPClient, error) {
	udpAddr, err := net.ResolveUDPAddr(udpNetwork, arg.Addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP(udpNetwork, nil, udpAddr)
	if err != nil {
		return nil, err
	}
	cli := &UDPClient{}
	sock := NewUDPSocket(ctx, UDPSocketArgs{isServer: false, conn: conn, onMsg: cli.udpOnMsg})
	cli.sock = sock

	subCtx, cancel := context.WithCancel(ctx)
	session := NewUDPSession(subCtx, UDPSessionArgs{
		cancel:       cancel,
		addr:         udpAddr,
		onMsg:        arg.OnMsg,
		onConnect:    arg.OnConnect,
		onDisconnect: arg.OnDisconnect,
		sendMsg:      cli.sock.sendMsg,
		now:          time.Now().Unix(),
	})
	cli.session = session
	return cli, nil
}

func (cli *UDPClient) udpOnMsg(ctx context.Context, msg []byte, addr *net.UDPAddr) {
	cli.session.recvMsg(msg, time.Now().Unix())
}

func (cli *UDPClient) SendMsg(ctx context.Context, msg []byte) error {
	return cli.sock.sendMsg(ctx, &udpDatagram{msg: msg, addr: cli.session.remoteDddr()})
}

func (cli *UDPClient) Close(ctx context.Context) {
	cli.session.Close(ctx)
	cli.sock.close(ctx)
}
