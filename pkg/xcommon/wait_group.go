package xcommon

import (
	"context"
	"gotu/pkg/xlog"
	"runtime/debug"
	"sync"
)

// 通过waitGroup控制协程
// defer wg.Done(), 不可在套一层func, recover不可跳过多层defer函数
type WaitGroup struct {
	w sync.WaitGroup
}

func (wg *WaitGroup) Add(n int) {
	wg.w.Add(n)
}

func (wg *WaitGroup) Done(ctx context.Context) {
	if r := recover(); r != nil {
		xlog.Get(ctx).Sugar().Errorf("Goroutine panic %v stack %v", r, string(debug.Stack()))
		panic(r)
	}
	wg.w.Done()
}

func (wg *WaitGroup) Wait() {
	wg.w.Wait()
}

// defer Recover(), 不可在套一层func, recover不可跳过多层defer函数
func Recover(ctx context.Context) {
	if r := recover(); r != nil {
		xlog.Get(ctx).Sugar().Errorf("Goroutine panic %v stack %v", r, string(debug.Stack()))
		panic(r)
	}
}
