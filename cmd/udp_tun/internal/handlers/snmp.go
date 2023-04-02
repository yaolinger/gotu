package handlers

import (
	"context"
	"gotu/pkg/xcommon"
	"sort"
	"sync"
)

var _tunSnmp = &tunSnmp{total: &singleTunSnmp{id: "total", base: newTunSnmpBase()}, snmps: make(map[string]*singleTunSnmp)}

type latencySlice struct {
	data []int64
}

func (ls *latencySlice) Len() int {
	return len(ls.data)
}

func (ls *latencySlice) Less(i, j int) bool {
	return ls.data[i] < ls.data[j]
}

func (ls *latencySlice) Swap(i, j int) {
	temp := ls.data[i]
	ls.data[i] = ls.data[j]
	ls.data[j] = temp
}

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
	recvClientBytes   uint64        // 来自客户端流量
	recvClientPackets uint64        // 来自客户端数据包数量
	recvProxyBytes    uint64        // 来自代理流量
	recvProxyPackets  uint64        // 来自代理数据包数量
	proxyLatencys     *latencySlice // 代理间通信延迟
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
	return &tunSnmpBase{proxyLatencys: &latencySlice{data: make([]int64, 0)}}
}

func (sts *singleTunSnmp) clone() *tunSnmpBase {
	sts.mu.Lock()
	defer sts.mu.Unlock()
	base := &tunSnmpBase{
		recvClientBytes:   sts.base.recvClientBytes,
		recvClientPackets: sts.base.recvClientPackets,
		recvProxyBytes:    sts.base.recvProxyBytes,
		recvProxyPackets:  sts.base.recvProxyPackets,
		proxyLatencys:     &latencySlice{data: make([]int64, 0)},
	}
	base.proxyLatencys.data = append(base.proxyLatencys.data, sts.base.proxyLatencys.data...)
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
	sts.base.proxyLatencys.data = append(sts.base.proxyLatencys.data, latency)
}

func (sts *singleTunSnmp) add(s *tunSnmpBase) {
	sts.mu.Lock()
	defer sts.mu.Unlock()
	sts.base.recvClientBytes += s.recvClientBytes
	sts.base.recvClientPackets += s.recvClientPackets
	sts.base.recvProxyBytes += s.recvProxyBytes
	sts.base.recvProxyPackets += s.recvProxyPackets
	sts.base.proxyLatencys.data = append(sts.base.proxyLatencys.data, s.proxyLatencys.data...)
}

func (sts *singleTunSnmp) close(ctx context.Context) {
	_tunSnmp.total.add(sts.clone())

	xcommon.PrintTable(ctx, []string{"udp(c:client,p:proxy)", "c2p-bytes(B)", "c2p-packets", "p2c-bytes(B)", "p2c-packets", "p-delay-90%(ms)", "p-delay-95%(ms)", "p-delay-99%(ms)", "p-delay-100%(ms)"}, [][]string{sts.values(ctx), _tunSnmp.total.values(ctx)})

	_tunSnmp.mu.Lock()
	delete(_tunSnmp.snmps, sts.id)
	_tunSnmp.mu.Unlock()
}

func (sts *singleTunSnmp) values(ctx context.Context) []string {
	strs := make([]string, 0)
	strs = append(strs, sts.id)

	var delay_100 int64
	var delay_99 int64
	var delay_95 int64
	var delay_90 int64

	sts.mu.Lock()
	defer sts.mu.Unlock()

	sort.Sort(sts.base.proxyLatencys)
	count_100 := len(sts.base.proxyLatencys.data)
	count_99 := count_100 * 99 / 100
	count_95 := count_100 * 95 / 100
	count_90 := count_100 * 90 / 100

	for i, delay := range sts.base.proxyLatencys.data {
		if i < count_100 {
			delay_100 += delay
		}
		if i < count_99 {
			delay_99 += delay
		}
		if i < count_95 {
			delay_95 += delay
		}
		if i < count_90 {
			delay_90 += delay
		}
	}
	average_100 := xcommon.SafeDivision(delay_100, int64(count_100))
	average_99 := xcommon.SafeDivision(delay_99, int64(count_99))
	average_95 := xcommon.SafeDivision(delay_95, int64(count_95))
	average_90 := xcommon.SafeDivision(delay_90, int64(count_90))

	strs = append(strs, xcommon.ToString(sts.base.recvClientBytes), xcommon.ToString(sts.base.recvClientPackets), xcommon.ToString(sts.base.recvProxyBytes), xcommon.ToString(sts.base.recvProxyPackets), xcommon.ToString(average_90), xcommon.ToString(average_95), xcommon.ToString(average_99), xcommon.ToString(average_100))
	return strs
}
