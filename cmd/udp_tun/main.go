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

var mode = flag.Bool("mode", true, "proxy mode[true:svr|false:cli]")

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

	reg := handlers.InitRegistry(*proxyAddr, *mode)

	svr, err := xnet.NewUDPServer(ctx, xnet.UDPSvrArgs{
		Addr:         *listenAddr,
		Timeout:      10,
		OnConnect:    reg.OnConnect,
		OnDisconnect: reg.OnDisconnect,
		OnMsg:        reg.OnMsg,
	},
	)
	if err != nil {
		panic(err)
	}
	defer svr.Close(ctx)

	xcommon.UntilSignal(ctx)
}
