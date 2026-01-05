package pes

import (
	"errors"
)

// Packet 구조체는 파싱된 PES 데이터를 담습니다.
type Packet struct {
	StreamID uint8  // 스트림 ID (e.g., 0xE0=Video, 0xC0=Audio)
	PTS      uint64 // Presentation Time Stamp (90kHz unit)
	DTS      uint64 // Decoding Time Stamp (90kHz unit)
	HasPTS   bool   // PTS 존재 여부
	HasDTS   bool   // DTS 존재 여부
	Payload  []byte // 실제 비디오/오디오 데이터 (ES)
}

// Parse 함수는 바이트 슬라이스를 받아 PES 패킷 정보를 반환합니다.
// 입력 Data는 00 00 01 (Start Code)로 시작해야 합니다.
func Parse(data []byte) (*Packet, error) {
	// 1. 최소 헤더 길이 체크 (6 bytes)
	if len(data) < 6 {
		return nil, errors.New("data too short for PES header")
	}

	// 2. Start Code Prefix 체크 (0x000001)
	if data[0] != 0x00 || data[1] != 0x00 || data[2] != 0x01 {
		return nil, errors.New("invalid PES start code prefix")
	}

	pkt := &Packet{
		StreamID: data[3],
	}

	// PES Packet Length (Big Endian)
	// packetLen := uint16(data[4]<<8 | uint16(data[5])
	// 비디오의 경우 0일 수 있으므로 (unbounded), 여기서는 큰 의미를 두지 않고 슬라이싱에 집중

	// 3. Optional Header 파싱 여부 결정
	// Program Stream Map, Padding Stream 등은 구조가 다름.
	// 일반적으로 Video(0xE0~0xEF), Audio(0xC0~0xDF)인 경우에만 확장 헤더를 파싱합니다.
	if isVideoOrAudio(pkt.StreamID) {
		if len(data) < 9 {
			return nil, errors.New("data too short for optional PES header")
		}

		// data[6]: '10' (2bit) + Scrambling control 등
		// data[7]: Flags (PTS/DTS flags 등)
		// data[8]: PES Header Data Length (확장 헤더의 나머지 길이)

		ptsDtsFlag := (data[7] >> 6) & 0x03 // 상위 2비트 추출
		headerDataLen := int(data[8])

		// 헤더가 끝나는 위치 = 기본헤더(6) + 확장헤더고정(3) + 확장헤더가변(HeaderDataLen)
		payloadStart := 9 + headerDataLen

		if len(data) < payloadStart {
			return nil, errors.New("buffer smaller than specified header length")
		}

		currentPos := 9

		// 4. PTS/DTS 파싱
		if ptsDtsFlag == 0x02 { // 10 : PTS only
			if headerDataLen >= 5 {
				pkt.PTS = parseTimestamp(data[currentPos : currentPos+5])
				pkt.HasPTS = true
			}
		} else if ptsDtsFlag == 0x03 { // 11 : PTS + DTS
			if headerDataLen >= 10 {
				pkt.PTS = parseTimestamp(data[currentPos : currentPos+5])
				pkt.HasPTS = true
				pkt.DTS = parseTimestamp(data[currentPos+5 : currentPos+10])
				pkt.HasDTS = true
			}
		}

		// 5. Payload 추출
		pkt.Payload = data[payloadStart:]
	} else {
		// 비디오/오디오가 아닌 경우 (예: padding) 단순 처리
		pkt.Payload = data[6:]
	}

	return pkt, nil
}

func isVideoOrAudio(id uint8) bool {
	// Audio: 0xC0 ~ 0xDF
	// Video: 0xE0 ~ 0xEF
	return (id >= 0xC0 && id <= 0xDF) || (id >= 0xE0 && id <= 0xEF)
}

func parseTimestamp(data []byte) uint64 {
	if len(data) < 5 {
		return 0
	}

	var ts uint64
	// 포맷: [0010] [32..30] [1] [29..15] [1] [14..0] [1]

	// Byte 0: 상위 4비트 마커 제거 -> 32..30 (3 bits)
	ts |= uint64(data[0]&0x0E) << 29
	// Byte 1: 29..22 (8 bits)
	ts |= uint64(data[1]) << 22
	// Byte 2: 하위 1비트 마커 제거 -> 21..15 (7 bits)
	ts |= uint64(data[2]&0xFE) << 14
	// Byte 3: 14..7 (8 bits)
	ts |= uint64(data[3]) << 7
	// Byte 4: 하위 1비트 마커 제거 -> 6..0 (7 bits)
	ts |= uint64(data[4]) >> 1

	return ts
}
