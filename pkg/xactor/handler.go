package xactor

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"gotu/pkg/xlog"
)

type (
	SyncHandler  func(ctx context.Context, req interface{}) (interface{}, error) // 同步handler
	AsyncHandler func(ctx context.Context, req interface{})                      // 异步handler
	TickHanler   func(ctx context.Context)                                       // 定时器handler
)

// M1 request
// M2 response
func SyncHandlerWrap[M1 any, M2 any](fn func(ctx context.Context, r *M1) (*M2, error)) SyncHandlerArgs {
	return SyncHandlerArgs{func(ctx context.Context, req interface{}) (interface{}, error) {
		r, ok := req.(*M1)
		if !ok {
			return nil, fmt.Errorf("sync handler req[%v] not type %v", req, reflect.TypeOf(new(M1)))
		}
		return fn(ctx, r)
	}, reflect.TypeOf(new(M1))}
}

// M1 request
func AsyncHandlerWrap[M1 any](fn func(ctx context.Context, r *M1)) AsyncHandlerArgs {
	return AsyncHandlerArgs{func(ctx context.Context, req interface{}) {
		r, ok := req.(*M1)
		if !ok {
			xlog.Get(ctx).Warn(fmt.Sprintf("async handler req[%v] not type %v", req, reflect.TypeOf(new(M1))))
			return
		}
		fn(ctx, r)
	}, reflect.TypeOf(new(M1))}
}

// handler管理
type actorHandler struct {
	syncHandlers   map[reflect.Type]SyncHandler
	asyncHandler   map[reflect.Type]AsyncHandler
	tickFns        []TickHanler // 定时任务
	tickerDuration time.Duration
}

type SyncHandlerArgs struct {
	H SyncHandler
	T reflect.Type
}

type AsyncHandlerArgs struct {
	H AsyncHandler
	T reflect.Type
}

type ActorHandlerArgs struct {
	Syncs          []SyncHandlerArgs  // 同步handlers(同步调用阻塞等待结果)
	Asyncs         []AsyncHandlerArgs // 异步handlers(异步调用不阻塞)
	Tickers        []TickHanler       // 定时handlers(定时callback)
	TickerDuration time.Duration      // 定时间隔(默认1 minute)
}

func newActorHandler(arg ActorHandlerArgs) (*actorHandler, error) {
	h := &actorHandler{
		syncHandlers:   make(map[reflect.Type]SyncHandler),
		asyncHandler:   make(map[reflect.Type]AsyncHandler),
		tickFns:        make([]TickHanler, 0),
		tickerDuration: arg.TickerDuration,
	}
	if h.tickerDuration == 0 {
		h.tickerDuration = defaultActorTickerDuration
	}
	for _, sync := range arg.Syncs {
		if h.syncHandlers[sync.T] != nil {
			return nil, fmt.Errorf("sync request[%v] is repeated", sync.T)
		}
		h.syncHandlers[sync.T] = sync.H
	}
	for _, async := range arg.Asyncs {
		if h.asyncHandler[async.T] != nil {
			return nil, fmt.Errorf("async request[%v] is repeated", async.T)
		}
		h.asyncHandler[async.T] = async.H
	}
	h.tickFns = append(h.tickFns, arg.Tickers...)
	return h, nil
}

func (handler *actorHandler) GetHandler() *actorHandler {
	return handler
}

func (handler *actorHandler) getSyncHandler(t reflect.Type) SyncHandler {
	return handler.syncHandlers[t]
}

func (handler *actorHandler) getAsyncHandler(t reflect.Type) AsyncHandler {
	return handler.asyncHandler[t]
}

func (handler *actorHandler) tick(ctx context.Context, state ActorState) {
	for _, fn := range handler.tickFns {
		fn(ctx)
	}
}
