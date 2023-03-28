package xcommon

import (
	"fmt"
	"reflect"
	"unsafe"
)

// 参考src/reflect/value.go
// 利用unfase.Pointer进行转换, 去除内存拷贝

func StringToBytes(str string) []byte {
	var b []byte
	l := len(str)
	p := (*reflect.SliceHeader)(unsafe.Pointer(&b))
	p.Data = (*reflect.StringHeader)(unsafe.Pointer(&str)).Data
	p.Len = l
	p.Cap = l
	return b
}

func BytesToString(b []byte) string {
	var s string
	hdr := (*reflect.StringHeader)(unsafe.Pointer(&s))
	hdr.Data = (*reflect.SliceHeader)(unsafe.Pointer(&b)).Data
	hdr.Len = len(b)
	return s
}

func ToString(v interface{}) string {
	return fmt.Sprintf("%v", v)
}
