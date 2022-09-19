package main

import (
	"net"
	"net/url"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

const bootEnodeUrl = "enode://22a8232c3abc76a16ae9d6c3b164f98775fe226f0917b0ca871128a74a8e9630b458460865bab457221f1d448dd9791d24c4e5d88786180ac185df813a68d4de@3.209.45.79:30303"

type LocalNode interface {
	GetPrivKeyBytes() []byte
	GetId() []byte
}

type LocalNodeData struct {
	privKey *secp256k1.PrivateKey
}

type RemoteNode struct {
	address *net.UDPAddr
}

type Enode struct {
	id      string
	host    string
	udpPort string
	tcpPort string
}

func NewLocalNode() (LocalNode, error) {
	key, err := secp256k1.GeneratePrivateKey()

	if err != nil {
		return nil, err
	}

	return LocalNodeData{
		privKey: key,
	}, nil
}

func (ln LocalNodeData) GetPrivKeyBytes() []byte {
	bytes := ln.privKey.Key.Bytes()
	return bytes[:]
}

func (ln LocalNodeData) GetId() []byte {
	// Index 0 is the uncrompressed serialized flag. Not needed.
	return ln.privKey.PubKey().SerializeUncompressed()[1:]
}

func ParseEnode(enodeUrl string) (*Enode, error) {
	u, err := url.Parse(enodeUrl)

	if err != nil {
		return nil, err
	}

	id := u.User.Username()
	host := u.Hostname()
	tcpPort := u.Port()
	udpPort := u.Query().Get("discport")

	if udpPort == "" {
		udpPort = tcpPort
	}

	return &Enode{id, host, tcpPort, udpPort}, nil
}

func GetBootNode() RemoteNode {
	bootEnode, _ := ParseEnode(bootEnodeUrl)

	address := bootEnode.host + ":" + bootEnode.udpPort
	udpAddress, _ := net.ResolveUDPAddr("udp4", address)

	return RemoteNode{
		address: udpAddress,
	}
}
