package xcommon_test

import (
	"context"
	"gonet/pkg/xcommon"
	"gonet/pkg/xlog"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"go.uber.org/zap"
)

func TestCast(t *testing.T) {
	ctx := context.Background()
	str := "string data"
	cbytes := xcommon.StringToBytes(str)
	cstr := xcommon.BytesToString(cbytes)

	xlog.Get(ctx).Info("String bytes better cast test.", zap.Any("base string", str), zap.Any("string", cstr), zap.Any("bytes", string(cbytes)))
	xlog.Get(ctx).Info("string bytes better cast test.", zap.Any("base string addr", (*reflect.StringHeader)(unsafe.Pointer(&str)).Data), zap.Any("string addr", (*reflect.StringHeader)(unsafe.Pointer(&cstr)).Data), zap.Any("bytes addr", (*reflect.SliceHeader)(unsafe.Pointer(&cbytes)).Data))

	// 标准转换, 存在内存拷贝
	stdbytes := []byte(str)
	stdstr := string(stdbytes)

	xlog.Get(ctx).Info("String bytes std cast test.", zap.Any("base string", str), zap.Any("string", stdstr), zap.Any("bytes", string(stdbytes)))
	xlog.Get(ctx).Info("string bytes std cast test.", zap.Any("base string addr", (*reflect.StringHeader)(unsafe.Pointer(&str)).Data), zap.Any("string addr", (*reflect.StringHeader)(unsafe.Pointer(&stdstr)).Data), zap.Any("bytes addr", (*reflect.SliceHeader)(unsafe.Pointer(&stdbytes)).Data))

	// 性能对比(4~5倍)
	times := 100000
	now := time.Now()
	for i := 0; i < times; i++ {
		stdbytes := []byte(str)
		_ = string(stdbytes)
	}
	xlog.Get(ctx).Info("Std string bytes cast", zap.Any("cost", time.Since(now)))

	now = time.Now()
	for i := 0; i < times; i++ {
		cbytes := xcommon.StringToBytes(str)
		_ = xcommon.BytesToString(cbytes)
	}
	xlog.Get(ctx).Info("Better string bytes cast", zap.Any("cost", time.Since(now)))
}
