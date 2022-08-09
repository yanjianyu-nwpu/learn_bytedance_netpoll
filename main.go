package main

import "net"

func main() {
	listener, err := net.Listen(network, address)
	if err != nil {
		panic("create net listener failed")
	}
	...
}