package xcommon

func SafeDivision[T ~int | int16 | int32 | int64 | uint16 | uint32 | uint64](a T, b T) T {
	if b == 0 {
		return 0
	}
	return a / b
}
