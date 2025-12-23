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
	ParseTsPacket    uint64
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

		}
	}
}
