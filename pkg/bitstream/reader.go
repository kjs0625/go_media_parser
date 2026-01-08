package bitstream

import (
	"fmt"
)

type Reader struct {
	data      []byte
	byteIdx   int
	bitOffset int // 0 ~ 7 (현재 바이트 내에서 읽은 비트 수)
}

func NewReader(data []byte) *Reader {
	return &Reader{data: data}
}

// read 1bit 0 or 1
func (r *Reader) ReadBit() (uint8, error) {
	if r.byteIdx >= len(r.data) {
		return 0, fmt.Errorf("EOF: not enough data")
	}

	b := r.data[r.byteIdx]
	shift := 7 - r.bitOffset
	bit := (b >> shift) & 0x01

	r.bitOffset++
	if r.bitOffset == 8 {
		r.bitOffset = 0
		r.byteIdx++
	}

	return bit, nil
}

// ReadBits: n비트를 읽어서 uint32로 반환 (최대 32비트)
func (r *Reader) ReadBits(n int) (uint32, error) {
	if n > 32 {
		return 0, fmt.Errorf("cannot read more than 32 bits at once")
	}

	var ret uint32
	for i := 0; i < n; i++ {
		bit, err := r.ReadBit()
		if err != nil {
			return 0, err
		}
		ret = (ret << 1) | uint32(bit)
	}
	return ret, nil
}

// ReadUE: Unsigned Exp-Golomb 코딩 읽기 (H.264/HEVC 등에서 사용)
func (r *Reader) ReadUE() (uint32, error) {
	leadingZeros := 0
	for {
		bit, err := r.ReadBit()
		if err != nil {
			return 0, err
		}
		if bit == 1 {
			break
		}
		leadingZeros++
	}

	if leadingZeros == 0 {
		return 0, nil
	}

	rest, err := r.ReadBits(leadingZeros)
	if err != nil {
		return 0, err
	}

	//
	return (1 << uint32(leadingZeros)) - 1 + rest, nil
}
