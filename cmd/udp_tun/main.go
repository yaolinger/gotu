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
var inLoss = flag.Int("inloss", 0, "in: loss packet 0~100")
var inLatency = flag.Int("inlatency", 0, "in: relay rand latency")
var outLoss = flag.Int("outloss", 0, "out: loss packet 0~100")
var outLatency = flag.Int("outlatency", 0, "out: relay rand latency")

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

	reg, err := handlers.InitRegistry(ctx, handlers.RegistryArgs{Addr: *proxyAddr, Mode: *mode, Header: *header, InLoss: uint32(*inLoss), InLatency: uint32(*inLatency), OutLoss: uint32(*outLoss), OutLatency: uint32(*outLatency)})
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
