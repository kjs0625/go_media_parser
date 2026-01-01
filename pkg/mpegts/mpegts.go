package mpegts

import (
	"fmt"
	"os"
	"path/filepath"
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
	HasAdaptationField bool
	AdaptationLength   uint8
	Discontinuity      bool
	RandomAccess       bool
	PCR                uint64
	HasPCR             bool

	// --- Payload ---
	Payload []byte
}

type Assembler struct {
	// PID별로 조립 중인 임시 버퍼
	buffers map[uint16][]byte

	// PID별로 열려 있는 파일 핸들 (계속 열고 닫으면 느리니까)
	files map[uint16]*os.File

	// 파일 저장할 폴더 경로
	outputDir string
}

// NewAssembler: 생성자
func NewAssembler(outputDir string) *Assembler {
	// 폴더가 없으면 생성
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		os.Mkdir(outputDir, 0755)
	}

	return &Assembler{
		buffers:   make(map[uint16][]byte),
		files:     make(map[uint16]*os.File),
		outputDir: outputDir,
	}
}

// AddPacket: 패킷을 받아서 조립 로직 수행
func (a *Assembler) AddPacket(pkt *TSPacket) {
	// Payload가 없거나, 이상한 패킷은 무시
	if len(pkt.Payload) == 0 {
		return
	}

	// PID가 0(PAT), 17(SDT) 등 메타데이터라면 조립하지 않고 패스 (선택 사항)
	// 여기서는 일단 모든 PID를 다 조립해봅니다.

	// PUSI (Payload Unit Start Indicator) 체크
	if pkt.PUSI {
		// 1. 기존에 조립하던 게 있었다면 저장 (Flush)
		if len(a.buffers[pkt.PID]) > 0 {
			a.saveToDisk(pkt.PID, a.buffers[pkt.PID])
		}

		// 2. 새로운 시작: 버퍼 초기화 및 현재 패킷 데이터 넣기
		// (새로운 슬라이스 할당해서 복사)
		a.buffers[pkt.PID] = append([]byte{}, pkt.Payload...)
	} else {
		a.buffers[pkt.PID] = append(a.buffers[pkt.PID], pkt.Payload...)
	}
}

func (a *Assembler) saveToDisk(pid uint16, data []byte) {
	if _, exists := a.files[pid]; !exists {
		filename := fmt.Sprintf("output_%d.h264", pid) // 일단 h264라고 가정
		path := filepath.Join(a.outputDir, filename)

		f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Printf("Error opening file for PID %d: %v\n", pid, err)
			return
		}
		a.files[pid] = f
		fmt.Printf("Created files: %s\n", filename)
	}

	// * 핵심: PES 헤더를 벗겨내고 순수 데이터(ES)만 추출 *
	esData := extractES(data)

	// 파일에 쓰기
	if len(esData) > 0 {
		_, err := a.files[pid].Write(esData)
		if err != nil {
			fmt.Printf("Error writing to PID %d: %v\n", pid, err)
		}
	}
}

func (a *Assembler) Close() {
	for pid, f := range a.files {
		fmt.Printf("Closing file for PID %d\n", pid)
		f.Close()
	}
}

// extractES : PES 패킷에서 헤더를 떼고 Payload(ES)만 반환
func extractES(pesData []byte) []byte {
	// 최소 PES 헤더 길이(6) + Optional Header 길이 필드 위치까지 확보
	if len(pesData) < 9 {
		return nil
	}

	// 1. Start Code 확인 (0x000001)
	if pesData[0] != 0x00 || pesData[1] != 0x00 || pesData[2] != 0x01 {
		// PES가 아닌 데이터(그냥 TS 조각 등)는 그냥 저장하거나 버림
		return pesData
	}

	// 2. Optional Header 길이 파악
	// PES 헤더 구조 : [StartCode 3][StreamID 1][PacketLen 2][Flags 2][HeaderLen 1][...Header Data...]
	// 8번 인덱스(9번째 바이트)가 "내 뒤에 헤더가 몇 바이트 더 있는지" 알려줌
	pesHeaderDataLen := int(pesData[8])

	// 3. 실제 데이터 시작 위치 계산
	// 기본 6바이트 + Flags(2바이트) + Len필드(1바이트) = 9바이트
	headerSize := 9 + pesHeaderDataLen

	if len(pesData) <= headerSize {
		return nil
	}

	// 헤더 뒤에 있는 진짜 데이터만 반환
	return pesData[headerSize:]
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
		pkt.HasAdaptationField = true
		afLen := data[4] // Adaption Filed Length
		pkt.AdaptationLength = afLen

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

func Print(p *TSPacket) {
	fmt.Printf("[TS] PID: %4d | PUSI: %-5v | CC: %2d | AFC: %d ", p.PID, p.PUSI, p.CC, p.AFC)

	afcDesc := ""
	switch p.AFC {
	case 2:
		afcDesc = "(Adpat)"
	case 3:
		afcDesc = "(Mix)"
	}
	fmt.Printf("%s\n", afcDesc)

	if p.HasAdaptationField {
		fmt.Printf("    └─ Adaptation: Len=%d, RAI=%v", p.AdaptationLength, p.RandomAccess)
		if p.HasPCR {
			fmt.Printf(", PCR=%d", p.PCR)
		}
		fmt.Println()
	}

	if len(p.Payload) > 0 {
		previewLen := 8
		if len(p.Payload) < previewLen {
			previewLen = len(p.Payload)
		}
		fmt.Printf("    └─ Payload: %3d bytes [% X ...]\n", len(p.Payload), p.Payload[:previewLen])
	}
}
