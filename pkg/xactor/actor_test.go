package xactor_test

import (
	"context"
	"testing"
	"time"

	"gotu/pkg/xactor"
	"gotu/pkg/xlog"

	"go.uber.org/zap"
)

var LogicActorName string = "LogicActor"

// 业务模块
type LogicActor struct {
	id int32
}

func (da *LogicActor) InitArg() xactor.ActorHandlerArgs {
	return xactor.ActorHandlerArgs{
		Syncs:          []xactor.SyncHandlerArgs{xactor.SyncHandlerWrap(da.syncShowData)},
		Asyncs:         []xactor.AsyncHandlerArgs{xactor.AsyncHandlerWrap(da.asyncShowData)},
		Tickers:        []xactor.TickHanler{da.tick},
		TickerDuration: 50 * time.Millisecond,
	}
}

func (da *LogicActor) Close(ctx context.Context) {
	xlog.Get(ctx).Info("Close data actor")
}

func (da *LogicActor) Name() string {
	return LogicActorName
}

type ShowDataReq struct {
	Str string
}

type ShowDataResp struct {
	Str string
}

// 同步业务处理
func (da *LogicActor) syncShowData(ctx context.Context, req *ShowDataReq) (*ShowDataResp, error) {
	da.id++
	return &ShowDataResp{Str: req.Str}, nil
}

// 异步业务处理
func (da *LogicActor) asyncShowData(ctx context.Context, req *ShowDataReq) {
	da.id++
	xlog.Get(ctx).Info("Async req", zap.Any("req", req))
}

// 定时器业务处理
func (da *LogicActor) tick(ctx context.Context) {
	xlog.Get(ctx).Info("tick", zap.Any("id", da.id))
}

func TestActor(t *testing.T) {
	ctx := context.Background()

	// 注册actor
	if err := xactor.NewActorGroutine(ctx, &LogicActor{id: 0}); err != nil {
		panic(err)
	}

	// 同步请求
	resp, err := xactor.SyncRequest[ShowDataReq, ShowDataResp](ctx, LogicActorName, &ShowDataReq{"buginventors"})
	if err != nil {
		panic(err)
	}
	xlog.Get(ctx).Info("Sync request", zap.Any("resp", resp))

	// 异步请求
	xactor.AsyncRequest(ctx, LogicActorName, &ShowDataReq{"buginventors"})

	time.Sleep(1 * time.Second)

	xactor.CloseAll(ctx)
}
