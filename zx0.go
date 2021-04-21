package zx0

import (
	"fmt"
)

const (
	InitialOffset = 1
	MaxScale      = 50
	QtyBlocks     = 1000
	MaxOffsetZx0  = 32640
	MaxOffsetZx7  = 2176
)

var (
//emptyBlock = Zx0Block{}
)

type Zx0 struct {
	outputData    []byte
	outputIndex   int
	inputIndex    int
	bitIndex      int
	bitMask       int
	diff          int
	backtrack     bool
	ghostRoot     *Zx0Block
	deadArray     []Zx0Block
	deadArraySize int
}

type Zx0Block struct {
	chain      *Zx0Block
	ghostChain *Zx0Block
	bits       int
	index      int
	offset     int
	length     int
	references int
}

func NewZx0() *Zx0 {
	return &Zx0{}
}

func (z *Zx0) Compress(optimal *Zx0Block, inputData []byte, input_size, skip, backwards_mode int, output_size *int, delta *int) []byte {
	var (
		next        *Zx0Block
		prev        *Zx0Block
		last_offset int  = InitialOffset
		first       bool = true
		i           int
	)
	/* calculate and allocate output buffer */
	*output_size = (optimal.bits + 18 + 7) / 8
	z.outputData = make([]byte, *output_size)

	/* initialize delta */
	z.diff = *output_size - input_size + skip
	*delta = 0

	/* un-reverse optimal sequence */
	next = nil
	for optimal != nil {
		prev = optimal.chain
		optimal.chain = next
		next = optimal
		optimal = prev
	}

	z.inputIndex = skip
	z.outputIndex = 0
	z.bitMask = 0

	for optimal = next.chain; optimal != nil; optimal = optimal.chain {
		if optimal.offset != 0 {
			/* copy literals indicator */
			if first {
				first = false
			} else {
				z.writeBit(0)
			}
			/* copy literals length */
			z.writeInterlacedEliasGamma(optimal.length, backwards_mode)

			/* copy literals values */
			for i = 0; i < optimal.length; i++ {
				z.writeByte(int(inputData[z.inputIndex]))
				z.readBytes(1, delta)
			}
		} else {
			if optimal.offset == last_offset {
				/* copy from last offset indicator */
				z.writeBit(0)

				/* copy from last offset length */
				z.writeInterlacedEliasGamma(optimal.length, backwards_mode)
				z.readBytes(optimal.length, delta)
			} else {
				/* copy from new offset indicator */
				z.writeBit(1)

				/* copy from new offset MSB */
				z.writeInterlacedEliasGamma((optimal.offset-1)/128+1, backwards_mode)

				/* copy from new offset LSB */
				if backwards_mode != 0 {
					z.writeByte(((optimal.offset - 1) % 128) << 1)
				} else {
					z.writeByte((255 - ((optimal.offset - 1) % 128)) << 1)
				}
				z.backtrack = true

				/* copy from new offset length */
				z.writeInterlacedEliasGamma(optimal.length-1, backwards_mode)
				z.readBytes(optimal.length, delta)

				last_offset = optimal.offset
			}
		}
	}

	/* end marker */
	z.writeBit(1)
	z.writeInterlacedEliasGamma(256, backwards_mode)

	return z.outputData
}

func (z *Zx0) readBytes(n int, delta *int) {
	z.inputIndex += n
	z.diff += n
	if z.diff > *delta {
		*delta = z.diff
	}
}

func (z *Zx0) writeByte(value int) {
	z.outputData[z.outputIndex] = byte(value)
	z.outputIndex++
	z.diff--
}

func (z *Zx0) writeBit(value int) {
	if z.backtrack {
		if value != 0 {
			z.outputData[z.outputIndex-1] |= 1
		}
		z.backtrack = false
	} else {
		if z.bitMask == 0 {
			z.bitMask = 128
			z.bitIndex = z.outputIndex
			z.writeByte(0)
		}
		if value != 0 {
			z.outputData[z.bitIndex] |= byte(z.bitMask)
		}
		z.bitMask >>= 1
	}
}

func (z *Zx0) writeInterlacedEliasGamma(value int, backwardsMode int) {
	var i int
	for i = 2; i <= value; i <<= 1 {
	}

	i >>= 1
	i >>= 1
	for i > 0 {
		z.writeBit(backwardsMode)
		z.writeBit(value & i)
		i >>= 1
	}
	if backwardsMode == 0 {
		z.writeBit(1)
	} else {
		z.writeBit(0)
	}
	//	z.writeBit(!backwardsMode)
}

func (z *Zx0) allocate(bits, index, offset, length int, chain *Zx0Block) *Zx0Block {
	var ptr *Zx0Block

	if z.ghostRoot != nil {
		ptr = z.ghostRoot
		z.ghostRoot = ptr.ghostChain
		if ptr.chain != nil {
			ptr.chain.references--
			if ptr.chain.references == 0 {
				ptr.chain.ghostChain = z.ghostRoot
				z.ghostRoot = ptr.chain
			}
		}
	} else {
		if z.deadArraySize == 0 {
			z.deadArray = make([]Zx0Block, QtyBlocks)
			z.deadArraySize = QtyBlocks
		}
		z.deadArraySize--
		ptr = &z.deadArray[z.deadArraySize]
	}
	ptr.bits = bits
	ptr.index = index
	ptr.offset = offset
	ptr.length = length
	if chain != nil {
		chain.references++
	}
	ptr.chain = chain
	ptr.references = 0
	return ptr
}

func (z *Zx0) Optimize(inputData []byte, inputSize, skip, offsetLimit int) *Zx0Block {
	var (
		lastLiteral    []*Zx0Block
		lastMatch      []*Zx0Block
		optimal        []*Zx0Block
		matchLength    []int
		bestLength     []int
		bestLengthSize int
		bits           int
		index          int
		offset         int
		length         int
		bits2          int
		dots           int = 2
		maxOffset      int = offsetCeiling(inputSize-1, offsetLimit)
	)
	/* allocate all main data structures at once */
	lastLiteral = make([]*Zx0Block, maxOffset+1)
	lastMatch = make([]*Zx0Block, maxOffset+1)

	optimal = make([]*Zx0Block, inputSize+1)
	matchLength = make([]int, maxOffset+1)
	bestLength = make([]int, inputSize+1)

	bestLength[2] = 2

	/* start with fake block */

	lastMatch[InitialOffset] = z.assign(lastMatch[InitialOffset], z.allocate(-1, skip-1, InitialOffset, 0, nil))

	fmt.Printf("[")

	/* process remaining bytes */
	for index = skip; index < inputSize; index++ {
		bestLengthSize = 2
		maxOffset = offsetCeiling(index, offsetLimit)
		for offset = 1; offset <= maxOffset; offset++ {
			if index != skip && index >= offset && inputData[index] == inputData[index-offset] {
				/* copy from last offset */
				if lastLiteral[offset] != nil {
					length = index - lastLiteral[offset].index
					bits = lastLiteral[offset].bits + 1 + eliasGammaBits(length)
					lastMatch[offset] = z.assign(lastMatch[offset],
						z.allocate(bits, index, offset, length, lastLiteral[offset]))
					if optimal[index] == nil || optimal[index].bits > bits {
						optimal[index] = z.assign(optimal[index], lastMatch[offset])
					}
				}
				/* copy from new offset */
				matchLength[offset]++
				if matchLength[offset] > 1 {
					if bestLengthSize < matchLength[offset] {
						bits = optimal[index-bestLength[bestLengthSize]].bits +
							eliasGammaBits(bestLength[bestLengthSize]-1)
						for {
							bestLengthSize++
							bits2 = optimal[index-bestLengthSize].bits + eliasGammaBits(bestLengthSize-1)
							if bits2 <= bits {
								bestLength[bestLengthSize] = bestLengthSize
								bits = bits2
							} else {
								bestLength[bestLengthSize] = bestLength[bestLengthSize-1]
							}
							if bestLengthSize >= matchLength[offset] {
								break
							}
						}
					}
					length = bestLength[matchLength[offset]]
					bits = optimal[index-length].bits + 8 + eliasGammaBits((offset-1)/128+1) +
						eliasGammaBits(length-1)
					if lastMatch[offset] == nil || lastMatch[offset].index != index || lastMatch[offset].bits > bits {
						lastMatch[offset] = z.assign(lastMatch[offset],
							z.allocate(bits, index, offset, length, optimal[index-length]))
						if optimal[index] == nil || optimal[index].bits > bits {
							optimal[index] = z.assign(optimal[index], lastMatch[offset])
						}
					}
				}
			} else {
				/* copy literals */
				matchLength[offset] = 0
				if lastMatch[offset] != nil {
					length = index - lastMatch[offset].index
					bits = lastMatch[offset].bits + 1 + eliasGammaBits(length) + length*8
					lastLiteral[offset] = z.assign(lastLiteral[offset],
						z.allocate(bits, index, 0, length, lastMatch[offset]))
					if optimal[index] == nil || optimal[index].bits > bits {
						optimal[index] = z.assign(optimal[index], lastLiteral[offset])
					}
				}
			}
		}

		if index*MaxScale/inputSize > dots {
			fmt.Printf(".")
			dots++
		}
	}

	fmt.Printf("]\n")

	return optimal[inputSize-1]
}

func offsetCeiling(index, offsetLimit int) int {
	if index > offsetLimit {
		return offsetLimit
	}
	if index < InitialOffset {
		return InitialOffset
	}
	return index
}

func eliasGammaBits(value int) int {
	var bits int = 1
	for value > 1 {
		bits += 2
		value >>= 1
	}
	return bits
}

func (z *Zx0) assign(ptr *Zx0Block, chain *Zx0Block) *Zx0Block {
	(*chain).references++
	if ptr != nil {
		(*ptr).references--
		if (*ptr).references == 0 {
			(*ptr).ghostChain = z.ghostRoot
			z.ghostRoot = ptr
		}
	}
	ptr = chain
	return ptr
}
