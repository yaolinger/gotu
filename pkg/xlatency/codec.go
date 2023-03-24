package xlatency

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"time"
)

var headerSizeof = binary.Size(latencyHeader{})

type latencyHeader struct {
	now int64
}

type latencyData struct {
	header *latencyHeader
	msg    []byte
}

func pack(ctx context.Context, msg []byte) ([]byte, error) {
	header := &latencyHeader{now: time.Now().UnixMilli()}
	ioWrite := bytes.NewBuffer(nil)
	if err := binary.Write(ioWrite, binary.LittleEndian, header); err != nil {
		return nil, err
	}
	msg = append(ioWrite.Bytes(), msg...)
	return msg, nil
}

func unpack(ctx context.Context, msg []byte) (*latencyData, error) {
	if len(msg) < headerSizeof {
		return nil, fmt.Errorf("msg %v not enough %v", len(msg), headerSizeof)
	}
	header := &latencyHeader{}
	ioReader := bytes.NewReader(msg[0:headerSizeof])
	if err := binary.Read(ioReader, binary.LittleEndian, header); err != nil {
		return nil, err
	}
	return &latencyData{header: header, msg: msg[headerSizeof:]}, nil
}
