package xnet

import "sync"

type bufferManager struct {
	pool *sync.Pool // buf对象池
}

func newBufferManager() *bufferManager {
	return &bufferManager{
		pool: &sync.Pool{
			New: func() interface{} {
				bs := make([]byte, readBufferSize)
				return &bs
			},
		},
	}
}

func (mgr *bufferManager) newBufferPool() *bufferPool {
	return &bufferPool{
		pool:  mgr.pool,
		cache: make([]byte, 0),
	}
}

// 非线程安全, 仅限单协程使用(read loop)
type bufferPool struct {
	pool  *sync.Pool // buf对象池
	cache []byte
}

func (bp *bufferPool) put(cache []byte) {
	bp.cache = append(bp.cache, cache...)
	for len(bp.cache) >= readBufferSize {
		tempBuf := bp.cache[0:readBufferSize]
		bp.pool.Put(&tempBuf)
	}
}

func (bp *bufferPool) get() []byte {
	return *bp.pool.Get().(*[]byte)
}
