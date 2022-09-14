package main

import (
	"bytes"
	"encoding/binary"
	"errors"
)

type PacketType byte

// Packet types
const (
	InvalidPacketType PacketType = 0x00
	PingPacketType    PacketType = 0x01
	PongPacketType    PacketType = 0x02
)

// Errors
var (
	ErrorPacketTooSmall    = errors.New("Packet too small")
	ErrorInvalidHash       = errors.New("Invalid hash")
	ErrorInvalidPacketType = errors.New("Invalid packet type")
)

const (
	hashLength      = 32
	signatureLength = 65
	headerSize      = hashLength + signatureLength + 1
)

type PacketHeader struct {
	hash       []byte
	signature  []byte
	packetType PacketType
}

type Packet[PacketData any] struct {
	header PacketHeader
	data   PacketData
}

type Endpoint struct {
	ip      string
	udpPort int
	tcpPort int
}

type PingPacketData struct {
	version    int
	from       Endpoint
	to         Endpoint
	expiration uint64
	enrSeqNum  int
}

type PongPacketData struct {
	to         Endpoint
	pingHash   []byte
	expiration uint64
	enrSeqNum  int
}

func (p *PingPacketData) ToRLP() ([]byte, error) {
	return Encode([]any{
		p.version,
		[]any{p.from.ip, p.from.udpPort, p.from.tcpPort},
		[]any{p.to.ip, p.to.udpPort, p.to.tcpPort},
		p.expiration,
		p.enrSeqNum})
}

func (p *PongPacketData) ToRLP() ([]byte, error) {
	return Encode([]any{
		[]any{p.to.ip, p.to.udpPort, p.to.tcpPort},
		p.pingHash,
		p.expiration,
		p.enrSeqNum,
	})
}

func DecodePacket(data []byte) (*Packet[any], error) {
	if len(data) < headerSize+1 {
		return nil, ErrorPacketTooSmall
	}

	header, err := decodePacketHeader(data)

	if err != nil {
		return nil, err
	}

	var packetData any = nil
	err = nil
	packetDataBytes := data[headerSize:]

	switch header.packetType {
	case PingPacketType:
		packetData, err = decodePingPacketData(packetDataBytes)
	case PongPacketType:
		packetData, err = decodePongPacketData(packetDataBytes)
	default:
		err = ErrorInvalidPacketType
	}

	return &Packet[any]{
		header: *header,
		data:   packetData,
	}, err
}

func wrapInPacket(packetData []byte, pType PacketType, privKey []byte) ([]byte, []byte, error) {
	packetBytes := make([]byte, headerSize+len(packetData))
	packetBytes[headerSize-1] = byte(pType)
	copy(packetBytes[headerSize:], packetData)

	sig, err := Sign(Keccak256(packetBytes[headerSize-1:]), privKey)

	if err != nil {
		return nil, nil, err
	}

	copy(packetBytes[hashLength:], sig)

	hash := Keccak256(packetBytes[hashLength:])
	copy(packetBytes, hash)

	return packetBytes, hash, nil
}

func NewPingPacket(version int, from, to Endpoint, expiration uint64, enrSeqNum int, privKey []byte) ([]byte, []byte, error) {
	packetData := PingPacketData{
		version,
		from,
		to,
		expiration,
		enrSeqNum,
	}

	encodedPacketData, err := packetData.ToRLP()

	if err != nil {
		return nil, nil, err
	}

	return wrapInPacket(encodedPacketData, PingPacketType, privKey)
}

func NewPongPacket(to Endpoint, pingHash []byte, expiration uint64, enrSeqNum int, privKey []byte) ([]byte, []byte, error) {
	packetData := PongPacketData{
		to,
		pingHash,
		expiration,
		enrSeqNum,
	}

	encodedPacketData, err := packetData.ToRLP()

	if err != nil {
		return nil, nil, err
	}

	return wrapInPacket(encodedPacketData, PongPacketType, privKey)
}

func decodeEndpoint(data []any) Endpoint {
	return Endpoint{
		ip:      data[0].(string),
		udpPort: int(decodeUInt64([]byte(data[1].(string)))),
		tcpPort: int(decodeUInt64([]byte(data[2].(string)))),
	}
}

func decodePingPacketData(data []byte) (*PingPacketData, error) {
	decoded, err := Decode(data)

	if err != nil {
		return nil, err
	}

	decodedList := decoded.([]any)
	versionString := decodedList[0].(string)

	version := int(decodeUInt64([]byte(versionString)))
	from := decodeEndpoint(decodedList[1].([]any))
	to := decodeEndpoint(decodedList[2].([]any))
	expiration := decodeUInt64([]byte(decodedList[3].(string)))
	enrSeqNum := int(decodeUInt64([]byte(decodedList[4].(string))))

	return &PingPacketData{
		version,
		from,
		to,
		expiration,
		enrSeqNum,
	}, nil
}

func decodePacketType(t byte) PacketType {
	switch t {
	case 0x01:
		return PingPacketType
	case 0x02:
		return PongPacketType
	default:
		return InvalidPacketType
	}
}

func decodePacketHeader(packet []byte) (*PacketHeader, error) {
	header := PacketHeader{
		hash:       packet[:hashLength],
		signature:  packet[hashLength : hashLength+signatureLength],
		packetType: decodePacketType(packet[hashLength+signatureLength]),
	}

	expectedHash := Keccak256(packet[hashLength:])
	err := validateHeader(header, expectedHash)

	if err != nil {
		return nil, err
	} else {
		return &header, nil
	}
}

func validateHeader(header PacketHeader, expectedHash []byte) error {
	if header.packetType == InvalidPacketType {
		return ErrorInvalidPacketType
	}

	if !bytes.Equal(header.hash, expectedHash) {
		return ErrorInvalidHash
	}

	return nil
}

func decodePongPacketData(data []byte) (*PongPacketData, error) {
	decoded, err := Decode(data)

	if err != nil {
		return nil, err
	}

	decodedList := decoded.([]any)

	return &PongPacketData{
		to:         decodeEndpoint(decodedList[0].([]any)),
		pingHash:   []byte(decodedList[1].(string)),
		expiration: decodeUInt64([]byte(decodedList[2].(string))),
		enrSeqNum:  int(decodeUInt64([]byte(decodedList[3].(string)))),
	}, nil
}

func decodeUInt64(data []byte) uint64 {
	if len(data) == 1 {
		if data[0] == 0x80 {
			return 0
		} else {
			return uint64(data[0])
		}
	}

	buf := new(bytes.Buffer)

	for i := 0; i < 8-len(data); i++ {
		buf.WriteByte(0)
	}

	buf.Write(data)

	return binary.BigEndian.Uint64(buf.Bytes())
}
