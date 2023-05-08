package xregistry

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"gotu/pkg/xlog"
	"gotu/pkg/xmsg"
	"gotu/pkg/xnet"

	"go.uber.org/zap"
)

// handlers: cmd => func
var handlers = make(map[int32]HandleFunc)

type HandleFunc func(ctx context.Context, state *State, req []byte) error

// 注册回调
// 非线程安全，无锁处理, init内调用
func Register(cmd int32, fn HandleFunc) {
	if _, ok := handlers[cmd]; ok {
		panic(fmt.Sprintf("cmd[%d] is repeated.", cmd))
	}
	handlers[cmd] = fn
}

// 消息处理
func OnMsg(ctx context.Context, arg xmsg.MsgArgs) error {
	s := arg.State.(*State)

	// 业务处理
	handler := handlers[arg.Header.Cmd]
	if handler != nil {
		return handler(ctx, s, arg.Payload)
	} else {
		return fmt.Errorf("can not find handler[%d]", arg.Header.Cmd)
	}
}

// 建立连接
func OnConnect(ctx context.Context, sock xnet.Socket) interface{} {
	xlog.Get(ctx).Debug("Svr connect", zap.Any("addr", sock.RemoteAddr()))
	return &State{Sock: sock}
}

// 断开连接
func OnDisconnect(ctx context.Context, state interface{}) {
	// 可做一些逻辑层操作
	s := state.(*State)
	xlog.Get(ctx).Debug("Svr disconnect", zap.Any("addr", s.Sock.RemoteAddr()))
}

// 函数包装：logic handler => HandleFunc
func HandleWarp[M1 any](fn func(ctx context.Context, stats *State, req *M1) error) func(ctx context.Context, state *State, msg []byte) error {
	return func(ctx context.Context, state *State, msg []byte) error {
		req := new(M1)

		// 反序列化
		if err := Unmarshal(msg, req); err != nil {
			return err
		}

		// 执行handler
		return fn(ctx, state, req)
	}
}

// 序列化(个人选择序列化方案)
func Marshal(data interface{}) ([]byte, error) {
	// TODO: proto格式检测, 序列化
	// pdata, ok := interface{}(data).(protoreflect.ProtoMessage)
	// if !ok {
	// 	return nil, fmt.Errorf("data[%v] not proto", data)
	// }
	// return proto.Marshal(pdata)

	// 默认序列化方案, 不支持变长数据结构
	ioWrite := bytes.NewBuffer(nil)
	if err := binary.Write(ioWrite, binary.LittleEndian, data); err != nil {
		return nil, err
	}
	msg := ioWrite.Bytes()
	return msg, nil
}

// 反序列化(个人选择序列化方案)
func Unmarshal(msg []byte, data interface{}) error {
	// TODO: proto格式检测,反序列化
	// pdata, ok := data.(protoreflect.ProtoMessage)
	// if !ok {
	// 	return fmt.Errorf("data[%v] not proto", pdata)
	// }
	// if err := proto.Unmarshal(msg, pdata); err != nil {
	// 	return err
	// }

	// 默认序列化方案, 不支持变长数据结构
	ioReader := bytes.NewReader(msg)
	if err := binary.Read(ioReader, binary.LittleEndian, data); err != nil {
		return err
	}
	return nil
}
