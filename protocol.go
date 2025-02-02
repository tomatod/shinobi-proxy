package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"errors"
	"net"
)

type IPv4Header struct {
	VersionIHL     uint8
	TOS            uint8
	TotalLength    uint16
	Identification uint16
	FlagsFragment  uint16
	TTL            uint8
	Protocol       uint8
	HeaderChecksum uint16
	SourceIP       [4]byte
	DestinationIP  [4]byte
}

type IPv4 struct {
	Header  *IPv4Header
	options []byte
	Payload []byte
}

func IPv4Parse(data []byte) (*IPv4, error) {
	var err error
	all := IPv4{}
	all.Header, err = IPv4HeaderParse(data)
	if err != nil {
		return nil, err
	}
	all.options = data[20:all.Header.HeaderLen()]
	all.Payload = data[all.Header.HeaderLen():]
	return &all, nil
}

func (i *IPv4) Bytes() ([]byte, error) {
	hBytes, err := i.Header.Bytes()
	if err != nil {
		return nil, err
	}
	return append(append(hBytes, i.options...), i.Payload...), nil
}

func IPv4HeaderParse(data []byte) (*IPv4Header, error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("%d byte data is too short.", len(data))
	}

	header := &IPv4Header{}
	reader := bytes.NewReader(data[:20])
	if err := binary.Read(reader, binary.BigEndian, header); err != nil {
		return nil, fmt.Errorf("failed to parse data to IPv4 header with error: %v", err)
	}
	if err := header.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate IPv4 header with error: %v", err)
	}

	return header, nil
}

// will be used (not use this now)
func (h *IPv4Header) SetChecksum() error {
	bytes, err := h.Bytes()
	if err != nil {
		return fmt.Errorf("failed to caliculate checksum: ", err)
	}
	copy(bytes[10:12], []byte{0, 0})
	h.HeaderChecksum = Checksum16Bits(bytes)
	return nil
}

func (h *IPv4Header) Validate() error {
	if h.Version() != 4 {
		return errors.New("only support version 4.")
	}
	if h.HeaderLen() < 20 {
		return errors.New("IPv4 header length must be more than 20 byte.")
	}
	return nil
}

func (h *IPv4Header) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, h)
	return buf.Bytes(), err
}

func (h *IPv4Header) HeaderLen() int {
	return int(h.VersionIHL & 0x0F) * 4
}

func (h *IPv4Header) TotalLen() int {
	return int(h.TotalLength) * 4
}

func (h *IPv4Header) Version() int {
	return int(h.VersionIHL >> 4)
}

func (h *IPv4Header) SrcIP() net.IP {
	return net.IP(([]byte)(h.SourceIP[:]))
}

func (h *IPv4Header) DstIP() net.IP {
	return net.IP(([]byte)(h.DestinationIP[:]))
}

func (h *IPv4Header) SetSrcIP(ip net.IP) {
	h.SourceIP = *(*[4]byte)(ip.To4())
}

func (h *IPv4Header) SetDstIP(ip net.IP) {
	h.DestinationIP = *(*[4]byte)(ip.To4())
}

func (h *IPv4Header) ProtoNum() int {
	return int(h.Protocol)
}

// will be used (not use this now)
type TCPHeader struct {
	SourcePort      uint16
	DestinationPort uint16
	SequenceNumber  uint32
	AckNumber       uint32
	DataOffsetFlags uint16
	WindowSize      uint16
	Checksum        uint16
	UrgentPointer   uint16
}

func TCPHeaderParse(data []byte) (*TCPHeader, error) {
	if len(data) < 20 {
		return nil, fmt.Errorf("%d byte data is too short.", len(data))
	}

	header := &TCPHeader{}
	reader := bytes.NewReader(data[:20])
	if err := binary.Read(reader, binary.BigEndian, header); err != nil {
		return nil, fmt.Errorf("failed to parse data to TCP header with error: ", err)
	}

	return header, nil
}

func(h *TCPHeader) Bytes() ([]byte, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, h)
	return buf.Bytes(), err
}

func(h *TCPHeader) HeaderLen() int {
	return int(h.DataOffsetFlags >> 12) * 4
}

