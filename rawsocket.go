package main

import (
	"net"
	"os"
	"syscall"
)

const (
	ETH_SIZE = 14
	MTU      = 9000
)

func CreateRawSocket(nicName string) (*os.File, error) {
	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(Htons(syscall.ETH_P_ALL)))
	if err != nil {
		return nil, err
	}

	inf, err := net.InterfaceByName(nicName)
	if err != nil {
		return nil, err
	}
	if nicName == "lo" {
		inf.HardwareAddr = []byte{0, 0, 0, 0, 0, 0}
	}
	sockaddr := syscall.SockaddrLinklayer{
		Protocol: Htons(syscall.ETH_P_ALL),
		Halen:    6,
		Addr:     *(*[8]byte)(append(inf.HardwareAddr, 0, 0)),
		Ifindex:  inf.Index,
	}

	if err = syscall.Bind(fd, &sockaddr); err != nil {
		return nil, err
	}

	return os.NewFile(uintptr(fd), nicName), nil
}
