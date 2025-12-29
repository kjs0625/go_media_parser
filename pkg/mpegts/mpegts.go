package mpegts

import (
	"fmt"
)

const TS_PACKET_SIZE = 188

type TSPacket struct {
	// --- 4 Byte Header ---
	SyncByte uint8  // 0x47
	TEI      bool   // Transport Error Indicator
	PUSI     bool   // Payload Unit Start Indicator
	Priority bool   // Transport Priority
	PID      uint16 // Packet Identifier (v or a)
	TSC      uint8  // Transport Scrambling Control
	AFC      uint8  // Adaption Field Control (01:Payload, 10:Adaption, 11:Both)
	CC       uint8  // Continuity Counter

	// --- Adaption Field (Optional) ---
	HasAdaptionField bool
	AdaptionLength   uint8
	Discontinuity    bool
	RandomAccess     bool
	PCR              uint64
	HasPCR           bool

	// --- Payload ---
	Payload []byte
}

func ParseTsPacket(data []byte) (*TSPacket, error) {
	if len(data) != 188 {
		return nil, fmt.Errorf("invalid packet size: %d", len(data))
	}
	if data[0] != 0x47 {
		return nil, fmt.Errorf("invalid sync byte: 0x%X", data[0])
	}

	pkt := &TSPacket{}

	// --- 1. Fixed Header Parsing ( 4 Bytes ) ---
	pkt.SyncByte = data[0]

	// Byte 1: TEI(1) | PUSH(1) | Priority(1) | PID(5) + BYTE2
	pkt.TEI = (data[1] & 0x80) != 0
	pkt.PUSI = (data[1] & 0x40) != 0
	pkt.Priority = (data[1] & 0x20) != 0
	pkt.PID = (uint16(data[1]&0x1F) << 8) | uint16(data[2])

	// Byte 3: TSC(2) | AFC(2) | CC(4)
	pkt.TSC = (data[3] >> 6) & 0x03
	pkt.AFC = (data[3] >> 4) & 0x03
	pkt.CC = data[3] & 0x0F

	// --- 2. Adaption Field Parsing ---
	payloadOffset := 4
	if pkt.AFC == 2 || pkt.AFC == 3 {
		pkt.HasAdaptionField = true
		afLen := data[4] // Adaption Filed Length
		pkt.AdaptionLength = afLen

		if afLen > 0 {
			flags := data[5]
			// 5번째 바이트의 플래그들
			pkt.Discontinuity = (flags & 0x80) != 0 // D7 : Discontinuity indicator
			pkt.RandomAccess = (flags & 0x40) != 0  // D6: Random Access indicator

			pcrFlag := (flags & 0x10) != 0 // D4: PCR flag

			// PCR 파싱 (총 6바이트: Base 33bit + Reserved 6bit + Ext 9bit)
			if pcrFlag && afLen >= 7 {
				// PCR Base (33 bits) 추출 로직
				// data[6] ~ data[11] 사용
				var pcrBase uint64
				pcrBase = (uint64(data[6]) << 25) |
					(uint64(data[7]) << 17) |
					(uint64(data[8]) << 9) |
					(uint64(data[9]) << 1) |
					(uint64(data[10]) >> 7)

				// PCR Extension은 보통 계산에서 제외하거나 별도로 둠 (여기선 Base만)
				pkt.PCR = pcrBase
				pkt.HasPCR = true
			}
		}
		// Adaption Field 길이만큼 오프셋 이동 (+1은 길이 필드 자체 크기)
		payloadOffset += int(afLen) + 1
	}

	// --- 3. Payload Extraction ---
	// AFC가 1(01) 또는 3(11) 이어야 페이로드가 있음
	if (pkt.AFC == 1 || pkt.AFC == 3) && payloadOffset < 188 {
		pkt.Payload = data[payloadOffset:]
	}

	return pkt, nil
}
