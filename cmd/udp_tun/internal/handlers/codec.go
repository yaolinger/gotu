package handlers

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"gotu/pkg/xlog"
	"time"

	"go.uber.org/zap"
)

var headerSizeof = binary.Size(header{})

type header struct {
	Now int64
}

func pack(ctx context.Context, msg []byte) ([]byte, error) {
	header := &header{Now: time.Now().UnixMilli()}
	ioWrite := bytes.NewBuffer(nil)
	if err := binary.Write(ioWrite, binary.LittleEndian, header); err != nil {
		return nil, err
	}
	msg = append(ioWrite.Bytes(), msg...)
	xlog.Get(ctx).Debug("Pack header", zap.Any("now", header.Now))
	return msg, nil
}

func unpack(ctx context.Context, msg []byte) ([]byte, error) {
	if len(msg) < headerSizeof {
		return nil, fmt.Errorf("msg %v not enough %v", len(msg), headerSizeof)
	}
	header := &header{}
	ioReader := bytes.NewReader(msg[0:headerSizeof])
	if err := binary.Read(ioReader, binary.LittleEndian, header); err != nil {
		return nil, err
	}
	now := time.Now().UnixMilli()
	xlog.Get(ctx).Debug("Unpack header", zap.Any("now", header.Now), zap.Any("delay", now-header.Now))
	return msg[headerSizeof:], nil
}
