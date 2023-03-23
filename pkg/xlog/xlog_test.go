package xlog_test

import (
	"context"
	"gotu/pkg/xlog"
	"testing"

	"go.uber.org/zap"
)

func TestXLOG(t *testing.T) {

	ctx := context.Background()

	ctx = xlog.NewContext(ctx, zap.String("gotu", "golang-net"))
	xlog.Get(ctx).Debug("日志测试")
	ctx = xlog.NewContext(ctx, zap.String("author", "rabbit-tank"))
	xlog.Get(ctx).Info("日志测试")
	ctx = xlog.NewContext(ctx, zap.String("qq", "1127406486"))
	xlog.Get(ctx).Warn("日志测试")
	ctx = xlog.NewContext(ctx, zap.String("qq-groups", "707886394"))
	xlog.Get(ctx).Error("日志测试")
}
