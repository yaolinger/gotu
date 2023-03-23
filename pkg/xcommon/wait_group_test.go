package xcommon_test

import (
	"context"
	"gotu/pkg/xcommon"
	"testing"
)

func TestWaitGroup(t *testing.T) {
	ctx := context.Background()
	var wg xcommon.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done(ctx)
		// 可打开注释测试
		// var data map[int]string
		// data[1] = "2"
	}()
	wg.Wait()
}

func TestRecover(t *testing.T) {
	ctx := context.Background()
	defer xcommon.Recover(ctx)
	// 可打开注释测试
	// 	var data map[int]string
	// 	data[1] = "2"
}
