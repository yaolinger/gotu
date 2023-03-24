package handlers

import (
	"gotu/pkg/xlatency"
	"gotu/pkg/xnet"
)

type State struct {
	latency *xlatency.LatencyActor
	cli     *xnet.UDPClient
	svrSock xnet.Socket
}
