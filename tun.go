package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"unsafe"
	"net"
)

const (
	IFNAMSIZ     = 16         // max size of TUN device name
	TUNSETIFF    = 0x400454ca // creating TUN device for ioctl
	SIOCSIFADDR  = 0x8916     // setting IP to device for ioctl
	SIOCSIFFLAGS = 0x8914     // setting UP/DOWN of device for ioctl
	IFF_TUN      = 0x0001     // TUN device
	IFF_NO_PI    = 0x1000     // none pucket information
	IFF_UP       = 0x1
)

type ifreqTap struct {
	Name  [IFNAMSIZ]byte
	Flags uint16
}

func CreateTunDevice(name string, vip net.IP) (*os.File, error) {
	file, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
	defer func() {
		if file != nil && err != nil {
			file.Close()
		}
	}()

	if err != nil {
		return nil, fmt.Errorf("failed to open /dev/net/tun: %v", err)
	}

	// create a TUN device
	var ifrt ifreqTap
	copy(ifrt.Name[:], name[:])
	ifrt.Flags = IFF_TUN | IFF_NO_PI

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), uintptr(TUNSETIFF), uintptr(unsafe.Pointer(&ifrt)))
	if errno != 0 {
		return nil, fmt.Errorf("Failed to make TUN device: %v", errno)
	}

	// set socket to TUN device
	socketFd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, syscall.IPPROTO_IP)
	if err != nil {
		return nil, fmt.Errorf("Failed to open socket: %v", err)
	}

	// set TUN device up
	var ifru ifreqTap
	copy(ifru.Name[:], name[:])
	ifru.Flags = IFF_UP
	_, _, errno = syscall.Syscall(syscall.SYS_IOCTL, uintptr(socketFd), uintptr(SIOCSIFFLAGS), uintptr(unsafe.Pointer(&ifru)))
	if errno != 0 {
		return nil, fmt.Errorf("Failed to up %s: %v", name, errno)
	}

	// add route to TUN device
	// TODO: rewrite this code without command
	err = exec.Command("ip", "route", "add", vip.String(), "dev", name).Run()
	if err != nil {
		return nil, fmt.Errorf("Failed to add route: %v", err)
	}

	return file, nil
}
