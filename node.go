package main

import (
	"net"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
)

type LocalNode interface {
	GetPrivKeyBytes() []byte
}

type LocalNodeData struct {
	privKey *secp256k1.PrivateKey
}

type RemoteNode struct {
	address *net.UDPAddr
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
