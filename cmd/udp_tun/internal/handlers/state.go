package handlers

import (
	"gotu/pkg/xlatency"
	"gotu/pkg/xnet"
)

type State struct {
	latency *xlatency.LatencyMock
	cli     *xnet.UDPClient
	svrSock xnet.Socket
}
