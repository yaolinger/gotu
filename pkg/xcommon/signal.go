package xcommon

import (
	"context"
	"gotu/pkg/xlog"
	"os/signal"
	"syscall"
)

func UntilSignal(ctx context.Context) {
	// 信号监听
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	xlog.Get(ctx).Info("Recv exit signal")
}
