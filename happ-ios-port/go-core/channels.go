package main

const chanSize = 4096
const maxPkt = 65536

var (
	toGo   = make(chan []byte, chanSize) // Swift -> Go (inbound from device)
	fromGo = make(chan []byte, chanSize) // Go -> Swift (outbound to device)
)
