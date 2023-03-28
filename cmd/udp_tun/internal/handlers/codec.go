package handlers

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"time"
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
	return msg, nil
}

func unpack(ctx context.Context, msg []byte) (int64, []byte, error) {
	if len(msg) < headerSizeof {
		return 0, nil, fmt.Errorf("msg %v not enough %v", len(msg), headerSizeof)
	}
	header := &header{}
	ioReader := bytes.NewReader(msg[0:headerSizeof])
	if err := binary.Read(ioReader, binary.LittleEndian, header); err != nil {
		return 0, nil, err
	}
	now := time.Now().UnixMilli()
	return now - header.Now, msg[headerSizeof:], nil
}
