package global

import "math"

type HighSource struct{}

func (f *HighSource) Uint64() uint64 {
	return math.MaxUint64
}

type LowSource struct{}

func (f *LowSource) Uint64() uint64 {
	return 0
}
