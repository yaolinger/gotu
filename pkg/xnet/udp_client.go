package xnet

import (
	"context"
	"gotu/pkg/xcommon"
	"gotu/pkg/xlog"
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

type UDPCliArgs struct {
	Addr         string
	Timeout      int
	OnMsg        OnHandlerOnce
	OnConnect    OnConnect
	OnDisconnect OnDisconnect
}

type UDPClient struct {
	arg *UDPCliArgs

	biudp *builtInUDP
}

func NewUDPClient(ctx context.Context, arg UDPCliArgs) (*UDPClient, error) {
	cli := &UDPClient{arg: &arg}
	biudp, err := newBuiltInUDP(ctx, *cli.arg)
	if err != nil {
		return nil, err
	}
	cli.biudp = biudp
	return cli, nil
}

func (cli *UDPClient) Reconnect(ctx context.Context) error {
	cli.biudp.close(ctx)
	biudp, err := newBuiltInUDP(ctx, *cli.arg)
	if err != nil {
		return err
	}
	cli.biudp = biudp
	return err
}

func (cli *UDPClient) SendMsg(ctx context.Context, msg []byte) error {
	return cli.biudp.sendMsg(ctx, msg)
}

func (cli *UDPClient) Close(ctx context.Context) {
	cli.biudp.close(ctx)
}

// 内置udp
type builtInUDP struct {
	sock    *UDPSocket
	session *UDPSession

	closeOnce sync.Once
	closeCh   chan struct{}
	wg        xcommon.WaitGroup
}

func newBuiltInUDP(ctx context.Context, arg UDPCliArgs) (*builtInUDP, error) {
	udpAddr, err := net.ResolveUDPAddr(udpNetwork, arg.Addr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP(udpNetwork, nil, udpAddr)
	if err != nil {
		return nil, err
	}
	biudp := &builtInUDP{closeCh: make(chan struct{})}
	sock := NewUDPSocket(ctx, UDPSocketArgs{isServer: false, conn: conn, onMsg: biudp.udpOnMsg})
	biudp.sock = sock

	subCtx, cancel := context.WithCancel(ctx)
	session := NewUDPSession(subCtx, UDPSessionArgs{
		cancel:       cancel,
		addr:         udpAddr,
		local:        conn.LocalAddr(),
		onMsg:        arg.OnMsg,
		onConnect:    arg.OnConnect,
		onDisconnect: arg.OnDisconnect,
		sendMsg:      biudp.sock.sendMsg,
		now:          time.Now().Unix(),
	})
	biudp.session = session

	biudp.wg.Add(1)
	go biudp.checkLoop(ctx, arg.Timeout)

	xlog.Get(ctx).Info("UDP client start success.", zap.Any("addr", arg.Addr))
	return biudp, nil
}

func (biudp *builtInUDP) checkLoop(ctx context.Context, timeout int) {
	defer biudp.wg.Done(ctx)

	ticker := time.NewTicker(udpCheckDuration)
	defer ticker.Stop()

loop:
	for {
		select {
		case <-ticker.C:
		case <-biudp.closeCh:
			break loop
		}

		if biudp.session.getActiveAt() < time.Now().Unix()-int64(timeout) {
			biudp.forceClose(ctx)
			xlog.Get(ctx).Warn("UDP session timeout", zap.Any("id", addrToString(biudp.session.remoteAddr())))
			break
		}
	}
}

func (biudp *builtInUDP) udpOnMsg(ctx context.Context, msg []byte, addr *net.UDPAddr) {
	if err := biudp.session.recvMsg(msg, time.Now().Unix()); err != nil {
		xlog.Get(ctx).Warn("Session on msg failed.", zap.Any("err", err))
	}
}

func (biudp *builtInUDP) sendMsg(ctx context.Context, msg []byte) error {
	return biudp.sock.sendMsg(ctx, &udpDatagram{msg: msg, addr: biudp.session.remoteAddr()})
}

func (biudp *builtInUDP) close(ctx context.Context) {
	biudp.forceClose(ctx)
	biudp.wg.Wait()
}

func (biudp *builtInUDP) forceClose(ctx context.Context) {
	biudp.closeOnce.Do(func() {
		biudp.session.Close(ctx)
		biudp.sock.close(ctx)
		close(biudp.closeCh)
	})
}
