package xmsg

import (
	"bytes"
	"context"
	"encoding/binary"
	"gonet/pkg/xnet"
)

var HeaderSizeof = binary.Size(Header{})

type MsgArgs struct {
	State   interface{}
	Header  *Header
	Payload []byte
}

// 解析数据包
func ParseMsgWarp(fn func(ctx context.Context, arg MsgArgs) error) xnet.OnHandlerOnce {
	return xnet.OnHandlerOnce(func(ctx context.Context, state interface{}, msg []byte) (int, error) {
		// header解析
		if len(msg) < HeaderSizeof {
			return 0, nil
		}
		header := &Header{}
		ioReader := bytes.NewReader(msg[0:HeaderSizeof])
		if err := binary.Read(ioReader, binary.LittleEndian, header); err != nil {
			return 0, err
		}
		if len(msg) < HeaderSizeof+int(header.Len) {
			return 0, nil
		}
		payload := msg[HeaderSizeof : HeaderSizeof+int(header.Len)]
		err := fn(ctx, MsgArgs{State: state, Header: header, Payload: payload})
		return HeaderSizeof + int(header.Len), err
	})
}

type PackMsgArgs struct {
	Seq     int32
	Cmd     int32
	Flag    int32
	Payload []byte
}

// 打包数据
func PackMsg(ctx context.Context, arg PackMsgArgs) ([]byte, error) {
	header := &Header{Seq: arg.Seq, Cmd: arg.Cmd, Flag: arg.Flag, Len: int32(len(arg.Payload))}
	ioWrite := bytes.NewBuffer(nil)
	if err := binary.Write(ioWrite, binary.LittleEndian, header); err != nil {
		return nil, err
	}
	msg := ioWrite.Bytes()
	msg = append(msg, arg.Payload...)
	return msg, nil
}
