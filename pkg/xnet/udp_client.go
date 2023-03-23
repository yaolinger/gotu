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

type UDPCliArgs struct {
	Addr         string
	OnMsg        OnHandlerOnce
	OnConnect    OnConnect
	OnDisconnect OnDisconnect
}

type UDPClient struct {
	sock    *UDPSocket
	session *UDPSession

	closeOnce sync.Once
	closeCh   chan struct{}
	wg        xcommon.WaitGroup
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
	cli := &UDPClient{closeCh: make(chan struct{})}
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

	cli.wg.Add(1)
	go cli.checkLoop(ctx)
	return cli, nil
}

func (cli *UDPClient) checkLoop(ctx context.Context) {
	defer cli.wg.Done(ctx)

	ticker := time.NewTicker(udpCheckDuration)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-ticker.C:
		case <-cli.closeCh:
			break loop
		}

		if cli.session.getActiveAt() < time.Now().Unix()-udpSessionTimeout {
			cli.forceClose(ctx)
			xlog.Get(ctx).Warn("UDP session timeout")
			break
		}
	}
}

func (cli *UDPClient) udpOnMsg(ctx context.Context, msg []byte, addr *net.UDPAddr) {
	if err := cli.session.recvMsg(msg, time.Now().Unix()); err != nil {
		xlog.Get(ctx).Warn("Session on msg failed.", zap.Any("err", err))
	}
}

func (cli *UDPClient) SendMsg(ctx context.Context, msg []byte) error {
	return cli.sock.sendMsg(ctx, &udpDatagram{msg: msg, addr: cli.session.remoteDddr()})
}

func (cli *UDPClient) Close(ctx context.Context) {
	cli.forceClose(ctx)
	cli.wg.Wait()
}

func (cli *UDPClient) forceClose(ctx context.Context) {
	cli.closeOnce.Do(func() {
		cli.session.Close(ctx)
		cli.sock.close(ctx)
		close(cli.closeCh)
	})
}
