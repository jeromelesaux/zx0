package encode

import "github.com/jeromelesaux/zx0"

func Encode(buf []byte) []byte {
	var outputSize int
	var delta int
	skip := 0
	z := zx0.Zx0{}
	inputSize := len(buf)
	out := z.Compress(z.Optimize(buf, inputSize, skip, zx0.MaxOffsetZx0), buf, inputSize, skip, 0, &outputSize, &delta)
	return out[:outputSize]
}
