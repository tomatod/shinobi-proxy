package main

import (
	"encoding/binary"
	"net"
)

func Htons(i uint16) uint16 {
	return (i<<8)&0xff00 | i>>8
}

// it will be depricated
func Checksum(data []byte) []byte {
	if len(data)%2 != 0 {
		data = append(data, 0)
	}

	var sum uint32
	for i := 0; i < len(data); i += 2 {
		for j := 0; j < 2; j++ {
			sum += uint32(data[i+j]) << ((1 - j) * 8)
		}
	}

	var checksum uint16 = 0xffff - uint16(sum) - uint16(sum>>16)
	bytes := make([]byte, 2)
	binary.BigEndian.PutUint16(bytes, checksum)
	return bytes
}

func Checksum16Bits(data []byte) uint16 {
	if len(data)%2 != 0 {
		data = append(data, 0)
	}

	var sum uint32
	for i := 0; i < len(data); i += 2 {
		for j := 0; j < 2; j++ {
			sum += uint32(data[i+j]) << ((1 - j) * 8)
		}
	}

	return 0xffff - uint16(sum) - uint16(sum>>16)
}

// it will be depricated
func ChecksumEntireIPPacket(ipPacket []byte) {
	ipHeaderSize := ipPacket[0] & 0b00001111
	ipHeader := ipPacket[:4*ipHeaderSize]
	protocol := int(ipPacket[9])
	payload := ipPacket[4*ipHeaderSize:]

	// caliculate the new checksum of ICMP, TCP and UDP header because of packet's change.
	if protocol == 1 /* ICMP */ {
		copy(payload[2:4], []byte{0, 0})
		copy(payload[2:4], Checksum(payload))
	}
	if protocol == 6 /* TCP */ {
		// reset checksum
		copy(payload[16:18], []byte{0, 0})

		// the target of check sum for TCP is pseudo header + TCP header + TCP payload.
		// create pseudo header.
		origin := []byte{}
		origin = append(origin, ipPacket[12:16]...)                                              // src IP
		origin = append(origin, ipPacket[16:20]...)                                              // dst IP
		origin = append(origin, 0)                                                               // reserved zero
		origin = append(origin, 6)                                                               // protocol number
		origin = append(origin, []byte{byte(len(payload) >> 8), byte(len(payload) & 0x00FF)}...) // TCP length

		// TCP header + payload (included in the 'payload' valuable)
		origin = append(origin, payload...)

		copy(payload[16:18], Checksum(origin))
	}
	if protocol == 17 /* UDP */ {
		printf(3, "[chks] CHECKSUM FOR UDP IS NOT IMPLEMENTED\n")
	}

	// caliculate the new checksum of IP header. The target of check sum for IP is only IP header.
	copy(ipHeader[10:12], []byte{0, 0}) // reset checksum
	copy(ipHeader[10:12], Checksum(ipHeader))
}

func GetIPAddrFromInterface(inf *net.Interface) net.IP {
	var (
		addrs []net.Addr
		err   error
	)
	if addrs, err = inf.Addrs(); err != nil {
		printf(0, "[util] failed to get address from net.Interface %v\n", err)
		return nil
	}
	for _, addr := range addrs {
		if ipv4Addr := addr.(*net.IPNet).IP.To4(); ipv4Addr != nil {
			return ipv4Addr
		}
	}
	return nil
}

var protocols = map[int]string{
	1:  "ICMP",
	6:  "TCP",
	17: "UDP",
}

func protocolStr(num int) string {
	if name, ok := protocols[num]; ok {
		return name
	}
	return "UNKNOWN_PROTOCOL"
}
