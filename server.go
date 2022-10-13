package main

import (
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"time"
)

const (
	packetExpiration = 20 * time.Second
	maxDatagramSize  = 1280
	enrSeqNum        = 1
)

type Server interface {
	GetIP() string
	GetUdpPort() int
	GetTcpPort() int
	Start()
	WritePing(*RemoteNode, func(*PongPacketData)) error
	FindNeighbors(*RemoteNode) error
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

func getExpiration() uint64 { return uint64(time.Now().Add(packetExpiration).Unix()) }

func (s serverImpl) Start() {
	fmt.Println("Server starting.", s.ip, s.udpPort)
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

		bytes := make([]byte, numBytes)
		copy(bytes, buf)

		go s.handlePacket(bytes, from)
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

	case NeighborsPacketType:
		s.handleNeighborsPacket(
			&decodedPacket.header,
			decodedPacket.data.(*NeighborsPacketData),
			from)
	default:
		fmt.Println("Cannot handle packet with type", t)
	}
}

func (s serverImpl) handlePingPacket(header *PacketHeader, data *PingPacketData, from *net.UDPAddr) {
	fmt.Println("Replying to ping packet with hash", hex.EncodeToString(header.hash))
	pongPacket, _, err := NewPongPacket(data.from, header.hash, getExpiration(),
		enrSeqNum, s.localNode.GetPrivKeyBytes())

	if err != nil {
		fmt.Println("Failed to create pong response for ping packet", err)
		return
	}

	toIp := NormalizeIp(data.from.ip)
	toAddr, err := net.ResolveUDPAddr("udp4", toIp+":"+fmt.Sprint(data.from.udpPort))

	if err != nil {
		fmt.Println("Failed to create address for pong packet", err)
		return
	}

	_, err = s.udpSocket.WriteToUDP(pongPacket, toAddr)

	if err != nil {
		fmt.Println("Failed to write pong packet", err)
		return
	}

	fmt.Println("Responded to ping")
}

func (s serverImpl) handlePongPacket(header *PacketHeader, data *PongPacketData, from *net.UDPAddr) {
	fmt.Println("Handling pong packet with ping hash", hex.EncodeToString(data.pingHash))

	mapKey := string(data.pingHash)
	callback := s.pingCallbacks[mapKey]

	if callback == nil {
		fmt.Println("Failed to find callback for ping")
		return
	}

	callback(data)
	delete(s.pingCallbacks, mapKey)
}

func (s serverImpl) handleNeighborsPacket(header *PacketHeader, data *NeighborsPacketData, from *net.UDPAddr) {
	fmt.Println("Got neighbors", len(data.nodes))

	for _, node := range data.nodes {
		s.localNode.AddNeighborNode(Enode{
			id:      node.nodeId,
			host:    NormalizeIp(node.ip),
			udpPort: strconv.Itoa(int(node.udpPort)),
			tcpPort: strconv.Itoa(int(node.tcpPort)),
		})
	}
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
		getExpiration(),
		enrSeqNum,
		s.localNode.GetPrivKeyBytes(),
	)

	if err != nil {
		return err
	}

	_, err = s.udpSocket.WriteToUDP(pingPacket, to.address)

	if err != nil {
		return err
	}

	s.pingCallbacks[string(hash)] = callback

	fmt.Println("Wrote ping with hash", hex.EncodeToString(hash))

	return nil
}

func (s serverImpl) FindNeighbors(to *RemoteNode) error {
	fmt.Println("Writing find neighbors request")
	packet, _, err := NewFindNodePacket(s.localNode.GetId(), getExpiration(), s.localNode.GetPrivKeyBytes())

	if err != nil {
		return err
	}

	numBytesWritten, err := s.udpSocket.WriteToUDP(packet, to.address)

	if err != nil {
		return err
	}

	fmt.Println("Wrote find neighbors bytes", numBytesWritten)

	return nil
}
