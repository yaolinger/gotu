package main

import (
	"context"
	"flag"
	"fmt"
	"gotu/cmd/udp_tun/internal/handlers"
	"gotu/pkg/xcommon"
	"gotu/pkg/xnet"
)

var listenAddr = flag.String("listen", ":6000", "udp listen addr")
var proxyAddr = flag.String("proxy", ":5000", "udp proxy addr")
var mode = flag.Int("mode", 0, "proxy mode (0:normal|1:client|2:server)")
var header = flag.Bool("header", true, "mode client/server add header")

func main() {
	flag.Parse()

	ctx := context.Background()
	defer xcommon.Recover(ctx)

	if *listenAddr == "" {
		panic(fmt.Sprintf("listen[%v] invalid", *listenAddr))
	}
	if *proxyAddr == "" {
		panic(fmt.Sprintf("connect[%v] invalid", *proxyAddr))
	}

	reg, err := handlers.InitRegistry(ctx, *proxyAddr, *mode, *header)
	if err != nil {
		panic(err)
	}

	svr, err := xnet.NewUDPServer(ctx, xnet.UDPSvrArgs{
		Addr:         *listenAddr,
		Timeout:      10,
		OnConnect:    reg.OnConnect,
		OnDisconnect: reg.OnDisconnect,
		OnMsg:        reg.OnMsg,
	})
	if err != nil {
		panic(err)
	}
	defer svr.Close(ctx)

	xcommon.UntilSignal(ctx)
}
