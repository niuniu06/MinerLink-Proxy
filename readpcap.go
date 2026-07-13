package main

import (
	"fmt"
	"log"

	"github.com/google/gopacket"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/layers"
)

func main() {
	handle, err := pcap.OpenOffline("log/btcproxy_capture.pcap")
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
	for packet := range packetSource.Packets() {
		if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
			tcp, _ := tcpLayer.(*layers.TCP)
			if len(tcp.Payload) > 0 {
				payloadStr := string(tcp.Payload)
				if len(payloadStr) > 200 {
					payloadStr = payloadStr[:200] + "..."
				}
				fmt.Printf("[%v] SRC: %v:%v DST: %v:%v -> %s\n",
					packet.Metadata().Timestamp.Format("15:04:05.000"),
					packet.NetworkLayer().NetworkFlow().Src(), tcp.SrcPort,
					packet.NetworkLayer().NetworkFlow().Dst(), tcp.DstPort,
					payloadStr)
			}
		}
	}
}
