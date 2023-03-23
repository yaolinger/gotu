package xactor

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"gotu/pkg/xlog"

	"go.uber.org/zap"
)

// 模拟actor模式
// 特性:
//	 1.异步单协程处理
//	 2.同步无锁编码
//	 3.同步阻塞消息处理/异步消息处理
//   4.支持ticker
type ActorGroutine struct {
	state ActorState // 数据状态
	box   *mailBox   // 消息分发
	*actorHandler

	wg      sync.WaitGroup
	closeCh chan struct{}
}

func NewActorGroutine(ctx context.Context, state ActorState) error {
	handler, err := newActorHandler(state.InitArg())
	if err != nil {
		return err
	}
	actor := &ActorGroutine{
		state:        state,
		box:          newMailBox(),
		actorHandler: handler,
		closeCh:      make(chan struct{}),
	}

	// 注册actor
	if err := registerActor(actor); err != nil {
		return err
	}

	actor.wg.Add(1)
	go actor.logicLoop(ctx)
	return nil
}

// 业务循环
func (actor *ActorGroutine) logicLoop(ctx context.Context) {
	defer actor.wg.Done()

	ticker := time.NewTicker(actor.actorHandler.tickerDuration)
	defer ticker.Stop()

	defer func() {
		// 关闭业务模块
		actor.state.Close(ctx)
	}()

loop:
	for {
		select {
		case m := <-actor.box.recvMail():
			if m.t == syncMail {
				handler := actor.actorHandler.getSyncHandler(reflect.TypeOf(m.req))
				if handler != nil {
					resp, err := handler(m.ctx, m.req)
					m.resultCh <- &result{resp: resp, err: err}
				} else {
					m.resultCh <- &result{err: fmt.Errorf("handler is nil")}
				}
			} else if m.t == asyncMail {
				handler := actor.actorHandler.getAsyncHandler(reflect.TypeOf(m.req))
				if handler != nil {
					handler(ctx, m.req)
				} else {
					xlog.Get(ctx).Warn("Async handler is nil", zap.Any("req", reflect.TypeOf(m.req)))
				}
			} else {
				xlog.Get(ctx).Warn("Mail type invalid", zap.Any("type", m.t))
			}
		case <-ticker.C:
		case <-actor.closeCh:
			break loop
		}

		// 触发定时任务
		actor.actorHandler.tick(ctx, actor.state)
	}
}

// 同步请求
func (actor *ActorGroutine) syncRequest(ctx context.Context, req interface{}) (interface{}, error) {
	m := newMail(ctx, syncMail, req)
	actor.box.sendMail(m)
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("cancel request")
	case r := <-m.resultCh:
		return r.resp, r.err
	}
}

// 同步请求(模板)
func SyncRequest[M1 any, M2 any](ctx context.Context, name string, req *M1) (*M2, error) {
	actor, err := GetActor(name)
	if err != nil {
		return nil, err
	}
	result, err := actor.syncRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	if resp, ok := result.(*M2); !ok {
		return nil, fmt.Errorf("result [%v] not type [%v]", reflect.TypeOf(result), reflect.TypeOf(new(M2)))
	} else {
		return resp, nil
	}
}

// 异步请求
func (actor *ActorGroutine) asyncRequest(ctx context.Context, req interface{}) {
	m := newMail(ctx, asyncMail, req)
	actor.box.sendMail(m)
}

// 异步请求
func AsyncRequest(ctx context.Context, name string, req interface{}) {
	actor, err := GetActor(name)
	if err != nil {
		xlog.Get(ctx).Error("Actor not exist.", zap.Any("name", name))
		return
	}
	actor.asyncRequest(ctx, req)
}

// 关闭
func (actor *ActorGroutine) Close(ctx context.Context) {
	close(actor.closeCh)
	actor.wg.Wait()

	// 删除actor
	deregisterActor(actor)
}
