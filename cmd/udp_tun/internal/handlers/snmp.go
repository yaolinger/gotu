package handlers

import (
	"context"
	"gotu/pkg/xcommon"
	"sync"
)

var _tunSnmp = &tunSnmp{total: &singleTunSnmp{id: "total", base: newTunSnmpBase()}, snmps: make(map[string]*singleTunSnmp)}

type tunSnmp struct {
	total *singleTunSnmp

	mu    sync.Mutex
	snmps map[string]*singleTunSnmp
}

type singleTunSnmp struct {
	id string

	mu   sync.Mutex
	base *tunSnmpBase
}

type tunSnmpBase struct {
	recvClientBytes   uint64  // 来自客户端流量
	recvClientPackets uint64  // 来自客户端数据包数量
	recvProxyBytes    uint64  // 来自代理流量
	recvProxyPackets  uint64  // 来自代理数据包数量
	proxyLatencys     []int64 // 代理间通信延迟
}

func getSingleTunSnmp(id string) *singleTunSnmp {
	_tunSnmp.mu.Lock()
	defer _tunSnmp.mu.Unlock()
	if sts, ok := _tunSnmp.snmps[id]; ok {
		return sts
	}

	//  创建新对象
	sts := &singleTunSnmp{id: id, base: newTunSnmpBase()}
	_tunSnmp.snmps[id] = sts
	return sts
}

func newTunSnmpBase() *tunSnmpBase {
	return &tunSnmpBase{proxyLatencys: make([]int64, 0)}
}

func (sts *singleTunSnmp) clone() *tunSnmpBase {
	sts.mu.Lock()
	defer sts.mu.Unlock()
	base := &tunSnmpBase{
		recvClientBytes:   sts.base.recvClientBytes,
		recvClientPackets: sts.base.recvClientPackets,
		recvProxyBytes:    sts.base.recvProxyBytes,
		recvProxyPackets:  sts.base.recvProxyPackets,
	}
	base.proxyLatencys = append(base.proxyLatencys, sts.base.proxyLatencys...)
	return base
}

func (sts *singleTunSnmp) recvClient(len int) {
	sts.mu.Lock()
	defer sts.mu.Unlock()
	sts.base.recvClientBytes += uint64(len)
	sts.base.recvClientPackets += 1
}

func (sts *singleTunSnmp) recvProxy(len int) {
	sts.mu.Lock()
	defer sts.mu.Unlock()
	sts.base.recvProxyBytes += uint64(len)
	sts.base.recvProxyPackets += 1
}

func (sts *singleTunSnmp) addLatency(latency int64) {
	sts.mu.Lock()
	defer sts.mu.Unlock()
	sts.base.proxyLatencys = append(sts.base.proxyLatencys, latency)
}

func (sts *singleTunSnmp) add(s *tunSnmpBase) {
	sts.mu.Lock()
	defer sts.mu.Unlock()
	sts.base.recvClientBytes += s.recvClientBytes
	sts.base.recvClientPackets += s.recvClientPackets
	sts.base.recvProxyBytes += s.recvProxyBytes
	sts.base.recvProxyPackets += s.recvProxyPackets
	sts.base.proxyLatencys = append(sts.base.proxyLatencys, s.proxyLatencys...)
}

func (sts *singleTunSnmp) close(ctx context.Context) {
	_tunSnmp.total.add(sts.clone())

	xcommon.PrintTable(ctx, []string{"udp", "client-to-proxy-bytes(B)", "client-to-proxy-packets", "average-proxy(cs)-delay(ms)", "proxy-to-client-bytes(B)", "proxy-to-client-packets"}, [][]string{sts.values(ctx), _tunSnmp.total.values(ctx)})

	_tunSnmp.mu.Lock()
	delete(_tunSnmp.snmps, sts.id)
	_tunSnmp.mu.Unlock()
}

func (sts *singleTunSnmp) values(ctx context.Context) []string {
	strs := make([]string, 0)
	strs = append(strs, sts.id)

	var allDelay int64
	var average int64

	sts.mu.Lock()
	defer sts.mu.Unlock()
	for _, delay := range sts.base.proxyLatencys {
		allDelay += delay
	}
	count := len(sts.base.proxyLatencys)
	if count != 0 {
		average = allDelay / int64(count)
	}
	strs = append(strs, xcommon.ToString(sts.base.recvClientBytes), xcommon.ToString(sts.base.recvClientPackets), xcommon.ToString(average), xcommon.ToString(sts.base.recvProxyBytes), xcommon.ToString(sts.base.recvProxyPackets))
	return strs
}
