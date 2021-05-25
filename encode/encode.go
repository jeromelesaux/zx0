package encode

import (
	"fmt"

	"github.com/jeromelesaux/zx0"
)

func Encode(buf []byte) []byte {
	var outputSize int
	var delta int
	skip := 0
	z := zx0.Zx0{}
	inputSize := len(buf)
	out := z.Compress(z.Optimize(buf, inputSize, skip, zx0.MaxOffsetZx0), buf, inputSize, skip, 0, &outputSize, &delta)
	fmt.Printf("Size:%d, compressed:%d\n", len(buf), len(out))
	return out[:outputSize]
}
