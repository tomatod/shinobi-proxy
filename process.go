package main

import (
	"os"
	"net"
	"sync"
	"encoding/binary"
	"fmt"
)

const (
	CHANNEL_SIZE = 100
)

func readWriteNic(cnf *Config, wg *sync.WaitGroup, nic *os.File, in chan []byte, out chan []byte) {
	defer wg.Done()

	var (
		peerMacAddr []byte
		nicMacAddr  []byte
		wgSub sync.WaitGroup
	)

	inf, err := net.InterfaceByName(cnf.NicName)
	if err != nil {
		printf(3, "[nicr] cannot read NIC info: %v", err)
	}
	nicMacAddr = []byte(inf.HardwareAddr)

	// read loop
	wgSub.Add(1)
	go func() {
		defer wgSub.Done()
		for {
			buffer := make([]byte, MTU)
			n, err := nic.Read(buffer)
			if err != nil {
				printf(2, "[nicr] error reading from NIC: %v\n", err)
				continue
			}
			if len(buffer) < 14 {
				continue
			}
			// only support IPv4
			etherProtocolType := binary.BigEndian.Uint16(buffer[12:14])
			if etherProtocolType != 0x0800 {
				continue
			}

			ip, err := IPv4Parse(buffer[ETH_SIZE:n])
			if err != nil {
				printf(0, "[nicr] cannot parse IP packet with error: %v\n", err)
				continue
			}
			if ip.Header.Version() != 4 {
				continue
			}

			// check src IP
			if !cnf.RemoteIP.Equal(ip.Header.SrcIP()) {
				continue
			}

			// memorize peer MAC addr at first time.
			if peerMacAddr == nil {
				peerMacAddr = buffer[6:12]
			}

			// check dst IP (this computer's IP)
			if !cnf.ExternalIP.Equal(ip.Header.DstIP()) {
				continue
			}

			// check protocol number
			if ip.Header.ProtoNum() != 1 /* ICMP */ && ip.Header.ProtoNum() != cnf.Protocol  {
				continue
			}

			// check dst port (this computer's port)
			if ip.Header.ProtoNum() == 6 /* UDP */ || ip.Header.ProtoNum() == 17 /* TCP */ {
				dstPort := int(ip.Payload[2]) << 8 + int(ip.Payload[3])
				if dstPort != cnf.InternalPort {
					continue
				}
			}

			printf(0,"[nicr] receive a %s packet from %v to %v\n", protocolStr(ip.Header.ProtoNum()), ip.Header.SrcIP(), ip.Header.DstIP())

			ipPacket, err := ip.Bytes()
			if err != nil {
				printf(2, "[nicr] failed to convert IPv4 struct to bytes with error: %v", err)
				continue
			}
			in <- ipPacket
		}
	}()

	// write loop
	wgSub.Add(1)
	go func() {
		defer wgSub.Done()
		for {
			ipPacket := <- out
			ethernetHeader := append(append(peerMacAddr, nicMacAddr...), []byte{0x08, 0x00}...) // IPv4 type is 0x0800
			ethernetFrame := append(append(ethernetHeader, ipPacket...)) 
			nic.Write(ethernetFrame)
			printf(0, "[nicw] write a %s packet to nic\n", protocolStr(int(ipPacket[9])))
		}
	}()

	wgSub.Wait()
}

func readWriteTun(cnf *Config, wg *sync.WaitGroup, tun *os.File, in chan []byte, out chan []byte) {
	defer wg.Done()

	var wgSub sync.WaitGroup

	// read loop
	wgSub.Add(1)
	go func() {
		defer wgSub.Done()
		for {
			buffer := make([]byte, MTU)
			n, err := tun.Read(buffer)
			if err != nil {
				printf(2,"[tunr] Error reading from TUN: %v\n", err)
				continue
			}

			ip, err := IPv4Parse(buffer[:n])
			if err != nil {
				printf(0,"[tunr] failed to parse IP header with error: %v\n", err)
				continue
			}

			// check src IP
			if !cnf.InternalIP.Equal(ip.Header.SrcIP()) {
				continue
			}

			// check dst IP
			if !cnf.ProxyIP.Equal(ip.Header.DstIP()) {
				continue
			}

			// check src port
			if ip.Header.ProtoNum() == 6 /* TCP */ || ip.Header.ProtoNum() == 17 /* UDP */ {
				srcPort := int(ip.Payload[0]) << 8 + int(ip.Payload[1])
				if cnf.InternalPort != srcPort {
					continue
				}
			}

			printf(0, "[tunr] receive a %s packet from %s to %s\n", protocolStr(ip.Header.ProtoNum()), ip.Header.SrcIP().String(), ip.Header.DstIP().String())

			// rewrite src and dst IP address from TUN's IP to remote IP.
			ip.Header.SetSrcIP(cnf.ExternalIP)
			ip.Header.SetDstIP(cnf.RemoteIP)
			ipPacket, err := ip.Bytes()
			if err != nil {
				printf(2, "[tunr] failed to convert IPv4Header to bytes with error: %v\n", err)
			}

			// caliculate Checksum anew
			ChecksumEntireIPPacket(ipPacket)

			out <- ipPacket
		}
	}()

	// writer rutine for TUN device. This routine processes the bellow steps.
	// 1. read inbound IP packet from reader routine for a specified NIC through a channel.
	// 2. rewrite src IP address from remote IP to TUN's one.
	// 3. caliculate Checksum for the new packet (IP header, and TCP, UDP or ICMP header).
	// 4. write the new packet into the TUN device.
	wgSub.Add(1)
	go func() {
		defer wgSub.Done()
		for {
			// wait IP packet from read ENI routine
			ipPacket := <- in

			// rewrite src IP address from remote IP to TUN's IP.
			copy(ipPacket[12:16], cnf.ProxyIP.To4())
			copy(ipPacket[16:20], cnf.InternalIP.To4())

			// caliculate Checksum anew
			ChecksumEntireIPPacket(ipPacket)

			// write IP packet into tun device.
			tun.Write(ipPacket)
			printf(0, "[tunw] write a packet into tun device\n")
		}
	}()

	wgSub.Wait()
}

func Run(cnf *Config) error {
	// create a TUN device
	tun, err := CreateTunDevice(cnf.TunName)
	if err != nil {
		return fmt.Errorf("failed to create a TUN Device: %v\n", err)
	}
	defer tun.Close()
	printf(0, "TUN device %s created successfully.\n", cnf.TunName)

	// create a rawsocket
	nic, err := CreateRawSocket(cnf.NicName)
	if err != nil {
		return fmt.Errorf("failed to create a raw socket: %v\n", err)
	}
	defer nic.Close()
	printf(0, "NIC %s opened by rawsocket successfully.\n", cnf.NicName)

	// start tunneling
	var wg sync.WaitGroup
	in  := make(chan []byte, CHANNEL_SIZE)
	out := make(chan []byte, CHANNEL_SIZE)
	wg.Add(1)
	go readWriteTun(cnf, &wg, tun, in, out)
	wg.Add(1)
	go readWriteNic(cnf, &wg, nic, in, out)
	wg.Wait()

	return nil
}
