package main

import (
	"flag"
)

func main() {
	// Parse command line flags
	serverAddress := flag.String("ip", "0.0.0.0:0", "IP:Port for the server")
	flag.Parse()

	// Start local node and server
	localNode, _ := NewLocalNode()
	server, _ := NewServer(*serverAddress, localNode)
	server.Start()

	// Write ping to bootnode
	bootNode := GetBootNode()
	server.WritePing(&bootNode)

	select {}
}
