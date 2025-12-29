package hexview

import "fmt"

func Print(data []byte) {
	const rowSize = 16

	for i := 0; i < len(data); i += rowSize {
		end := i + rowSize
		if end > len(data) {
			end = len(data)
		}

		row := data[i:end]

		// 오프셋
		fmt.Printf("%04X ", i)

		// Hex
		for j := 0; j < rowSize; j++ {
			if j < len(row) {
				fmt.Printf("%02X ", row[j])
			} else {
				fmt.Print("   ") // 빈 공간 채우기
			}
			// 8바이트마다 중간 공백 추가 (가독성)
			if j == 7 {
				fmt.Print("  ")
			}
		}

		fmt.Print(" |")

		// ASCII
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
