package xmsg

// 请求头包
type Header struct {
	Seq  int32 // 客户端Seq, 递增
	Cmd  int32 // 协议号
	Flag int32 // 特殊标识(加密...)
	Len  int32 // 数据长度
}
