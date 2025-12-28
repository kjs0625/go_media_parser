package main

import (
	"fmt"
	"io"
	"os"
)

const (
	TS_PACKET_SIZE = 188
	MAX_PACKETS    = 100
)

func main() {
	filename := "sample1.ts"
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("파일을 열 수 없습니다: %v\n", err)
		return
	}
	defer file.Close()

	buffer := make([]byte, TS_PACKET_SIZE)
	packetCount := 0

	fmt.Printf(">> REading file: %s (Shwing first %d packets)\n\n", filename, MAX_PACKETS)

	for {
		n, err := io.ReadFull(file, buffer)
		if err == io.EOF {
			break
		}
		if err != nil && err != io.ErrUnexpectedEOF {
			panic(err)
		}
		if n < TS_PACKET_SIZE {
			fmt.Println("Warning: 마지막에 188바이트보다 작은 데이터가 남았습니다.")
			break
		}

		packetCount++

		PrintHexDump(buffer)

		if packetCount >= MAX_PACKETS {
			fmt.Println("...(생략)...")
			break
		}
	}

	//mpegts.ParseTsPacket()
}

func PrintHexDump(data []byte) {
	const rowSize = 16

	for i := 0; i < len(data); i += rowSize {
		end := i + rowSize
		if end > len(data) {
			end = len(data)
		}

		row := data[i:end]

		// 1. 오프셋 (주소) 출력
		fmt.Printf("%04X ", i)

		// 2. 16진수(Hex) 출력
		for j := 0; j < rowSize; j++ {
			if j < len(row) {
				fmt.Printf("%02X ", row[j])
			} else {
				fmt.Print("    ") // 빈 공간 채우기
			}
			// 8바이트마다 중간 공백 추가 (가독성)
			if j == 7 {
				fmt.Print("  ")
			}
		}

		fmt.Print(" |")

		// 3. ASCII 문자 출력 (읽을 수 있는 것만, 나머지는 .)
		for _, b := range row {
			if b >= 32 && b <= 126 {
				fmt.Printf("%c", b)
			} else {
				fmt.Print(".")
			}
		}
		fmt.Println("|")
	}
}
