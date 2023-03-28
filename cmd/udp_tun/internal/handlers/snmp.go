package handlers

import (
	"context"
	"fmt"
	"gotu/pkg/xlog"
	"sync"
)

var _tunSnmp = &tunSnmp{total: &singleTunSnmp{id: "total", inLatencys: make([]int64, 0)}, snmps: make(map[string]*singleTunSnmp)}

type tunSnmp struct {
	total *singleTunSnmp

	mu    sync.Mutex
	snmps map[string]*singleTunSnmp
}

type singleTunSnmp struct {
	id string

	inMu       sync.Mutex
	inBytes    uint64  // 入站流量
	inPackets  uint64  // 入站数据包数量
	inLatencys []int64 // 入站延迟

	outMu      sync.Mutex
	outBytes   uint64 // 出站流量
	outPackets uint64 // 出站数据包数量
}

type singleTunSnmpClone struct {
	inBytes    uint64  // 入站流量
	inPackets  uint64  // 入站数据包数量
	inLatencys []int64 // 入站延迟
	outBytes   uint64  // 出站流量
	outPackets uint64  // 出站数据包数量
}

func getSingleTunSnmp(id string) *singleTunSnmp {
	_tunSnmp.mu.Lock()
	defer _tunSnmp.mu.Unlock()
	if sts, ok := _tunSnmp.snmps[id]; ok {
		return sts
	}

	//  创建新对象
	sts := &singleTunSnmp{id: id, inLatencys: make([]int64, 0)}
	_tunSnmp.snmps[id] = sts
	return sts
}

func (sts *singleTunSnmp) clone() *singleTunSnmpClone {
	c := &singleTunSnmpClone{}

	sts.inMu.Lock()
	c.inBytes = sts.inBytes
	c.inPackets = sts.inPackets
	c.inLatencys = append(c.inLatencys, sts.inLatencys...)
	sts.inMu.Unlock()

	sts.outMu.Lock()
	c.outBytes = sts.outBytes
	c.outPackets = sts.outPackets
	sts.outMu.Unlock()

	return c
}

func (sts *singleTunSnmp) recv(len int, delay int64) {
	sts.inMu.Lock()
	defer sts.inMu.Unlock()
	sts.inBytes += uint64(len)
	sts.inPackets += 1
	sts.inLatencys = append(sts.inLatencys, delay)
}

func (sts *singleTunSnmp) send(len int) {
	sts.outMu.Lock()
	defer sts.outMu.Unlock()
	sts.outBytes += uint64(len)
	sts.outPackets += 1
}

func (sts *singleTunSnmp) add(s *singleTunSnmpClone) {
	sts.inMu.Lock()
	sts.inBytes += s.inBytes
	sts.inPackets += s.inPackets
	sts.inLatencys = append(sts.inLatencys, s.inLatencys...)
	sts.inMu.Unlock()

	sts.outMu.Lock()
	sts.outBytes += s.outBytes
	sts.outPackets += s.outPackets
	sts.outMu.Unlock()
}

func (sts *singleTunSnmp) close(ctx context.Context) {
	_tunSnmp.total.add(sts.clone())

	sts.display(ctx)
	_tunSnmp.total.display(ctx)

	_tunSnmp.mu.Lock()
	delete(_tunSnmp.snmps, sts.id)
	_tunSnmp.mu.Unlock()
}

func (sts *singleTunSnmp) display(ctx context.Context) {
	str := ""
	var allDelay int64
	var average int64

	sts.inMu.Lock()
	for _, delay := range sts.inLatencys {
		allDelay += delay
	}
	if sts.inPackets != 0 {
		average = allDelay / int64(sts.inPackets)
	}
	str = fmt.Sprintf("UDP inBytes[%vB] inPackets[%v] averageDelay[%vms]", sts.inBytes, sts.inPackets, average)
	sts.inMu.Unlock()

	sts.outMu.Lock()
	str = fmt.Sprintf("%s outBytets[%vB] outPackets[%v] by[%v]", str, sts.outBytes, sts.outPackets, sts.id)
	sts.outMu.Unlock()

	xlog.Get(ctx).Info(str)
}
