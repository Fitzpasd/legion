package main

import (
	"fmt"
	"net"
	"time"
)

const (
	packetExpiration = 20 * time.Second
	maxDatagramSize  = 1280
)

type Server interface {
	GetIP() string
	GetUdpPort() int
	GetTcpPort() int
	Start()
	WritePing(*RemoteNode, func(*PongPacketData)) error
}

type serverImpl struct {
	localNode     LocalNode
	udpSocket     *net.UDPConn
	ip            string
	udpPort       int
	tcpPort       int
	pingCallbacks map[string]func(*PongPacketData)
}

func NewServer(localAddress string, localNode LocalNode) (Server, error) {
	socket, err := net.ListenPacket("udp4", localAddress)

	if err != nil {
		return nil, err
	}

	usocket := socket.(*net.UDPConn)
	var ip string
	uaddr := socket.LocalAddr().(*net.UDPAddr)
	if uaddr.IP.IsUnspecified() {
		ip = "127.0.0.1"
	} else {
		ip = uaddr.IP.String()
	}

	udpPort := uaddr.Port

	return serverImpl{
		localNode,
		usocket,
		ip,
		udpPort,
		0,
		make(map[string]func(*PongPacketData)),
	}, nil
}

func (s serverImpl) GetIP() string   { return s.ip }
func (s serverImpl) GetUdpPort() int { return s.udpPort }
func (s serverImpl) GetTcpPort() int { return s.tcpPort }

func (s serverImpl) Start() {
	fmt.Println("Server starting.")
	go s.readLoop()
}

func (s serverImpl) readLoop() {
	buf := make([]byte, maxDatagramSize)
	for {
		numBytes, from, err := s.udpSocket.ReadFromUDP(buf)

		if err != nil {
			panic(err)
		}

		fmt.Println("Read new packet. Size", numBytes)
		s.handlePacket(buf[:numBytes], from)
	}
}

func (s serverImpl) handlePacket(packetBytes []byte, from *net.UDPAddr) {
	decodedPacket, err := DecodePacket(packetBytes)

	if err != nil {
		fmt.Println("Error handling packet", err)
		return
	}

	t := decodedPacket.header.packetType
	switch t {
	case PingPacketType:
		s.handlePingPacket(
			&decodedPacket.header,
			decodedPacket.data.(*PingPacketData),
			from)
	case PongPacketType:
		s.handlePongPacket(
			&decodedPacket.header,
			decodedPacket.data.(*PongPacketData),
			from)
	default:
		fmt.Println("Cannot handle packet with type", t)
	}
}

func (s serverImpl) handlePingPacket(header *PacketHeader, data *PingPacketData, from *net.UDPAddr) {
	fmt.Println("Handling ping packet")
}

func (s serverImpl) handlePongPacket(header *PacketHeader, data *PongPacketData, from *net.UDPAddr) {
	fmt.Println("Handling pong packet")

	mapKey := string(data.pingHash)
	callback := s.pingCallbacks[mapKey]

	if callback == nil {
		fmt.Println("Failed to find callback for ping")
		return
	}

	callback(data)
	delete(s.pingCallbacks, mapKey)
}

func (s serverImpl) WritePing(to *RemoteNode, callback func(*PongPacketData)) error {
	fmt.Println("Writing ping to", to.address.IP, to.address.Port)

	pingPacket, hash, err := NewPingPacket(4,
		Endpoint{
			s.GetIP(),
			s.GetUdpPort(),
			s.GetTcpPort(),
		},
		Endpoint{
			to.address.IP.To4().String(),
			to.address.Port,
			0,
		},
		uint64(time.Now().Add(packetExpiration).Unix()),
		1,
		s.localNode.GetPrivKeyBytes(),
	)

	if err != nil {
		return err
	}

	numBytesWritten, err := s.udpSocket.WriteToUDP(pingPacket, to.address)

	if err != nil {
		return err
	}

	s.pingCallbacks[string(hash)] = callback

	fmt.Println("Wrote ping bytes", numBytesWritten)

	return nil
}
