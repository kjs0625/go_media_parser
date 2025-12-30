package main

import (
	"fmt"
	"io"
	"os"

	"go_media_parser/pkg/mpegts"
)

const (
	TS_PACKET_SIZE = 188
	MAX_PACKETS    = 100
)

func main() {
	filename := "test.ts"
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("파일을 열 수 없습니다: %v\n", err)
		return
	}
	defer file.Close()

	buffer := make([]byte, TS_PACKET_SIZE)
	packetCount := 0

	fmt.Printf(">> Reading file: %s (Showing first %d packets)\n\n", filename, MAX_PACKETS)

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

		tsPacket, err := mpegts.ParseTsPacket(buffer)
		if err != nil {
			fmt.Println("failed to parse ts packet")
		} else {
			mpegts.Print(tsPacket)
		}

		//hexview.Print(buffer)

		if packetCount >= MAX_PACKETS {
			fmt.Println("...(생략)...")
			break
		}
	}

	//mpegts.ParseTsPacket()
}
