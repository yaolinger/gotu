package xactor

import (
	"context"
	"time"
)

const (
	syncMail  mailType = 0 // 同步mail
	asyncMail mailType = 1 // 异步mail
)

var (
	defaultActorTickerDuration = 1 * time.Minute // 默认ticker时长
	mailMaxCount               = 100             // 最大mail数量
)

type mailType int

// 模块
type ActorState interface {
	InitArg() ActorHandlerArgs // 初始化参数,用于注册
	Name() string              // 名称,模块名称
	Close(ctx context.Context) // 关闭
}
