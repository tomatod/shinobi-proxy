package main

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
)

type Config struct {
	NicName      string
	TunName      string
	InternalIP   net.IP
	ExternalIP   net.IP
	RemoteIP     net.IP
	ProxyIP      net.IP
	InternalPort int
	Protocol     int
}

func (c *Config) validate() error {
	if c.NicName == "" {
		return errors.New("NIC name must be specified by -e flag")
	}
	if c.RemoteIP == nil {
		return errors.New("remote IP must be specified by -r flag")
	}
	if c.InternalPort == 0 {
		return errors.New("port number must be specified by -p flag")
	}
	return nil
}

func commandInit(fs *flag.FlagSet, conf *Config, args []string) error {
	fs.StringVar(&conf.TunName, "d", "tun0", "name of TUN device for this program.")
	fs.IntVar(&conf.InternalPort, "p", 0, "port number.")

	conf.Protocol = 1
	fs.Func("t", "transport protocol TCP or UDP", func(proto string) error {
		switch proto {
		case "TCP", "tcp":
			conf.Protocol = 6
		case "UDP", "udp":
			conf.Protocol = 17
		case "ICMP", "icmp":
			conf.Protocol = 1
		default:
			return fmt.Errorf("%s is not acceptable.", proto)
		}
		return nil
	})

	fs.Func("e", "name of the NIC receiving access from remote.", func(nicName string) error {
		inf, err := net.InterfaceByName(nicName)
		if err != nil {
			return fmt.Errorf("failed to handle '%s' with error: %v", nicName, err)
		}
		conf.NicName = nicName
		conf.ExternalIP = GetIPAddrFromInterface(inf)
		if conf.ExternalIP == nil {
			return fmt.Errorf("failed to get IPv4 from '%s'", nicName, err)
		}
		if conf.InternalIP == nil {
			conf.InternalIP = conf.ExternalIP // set internal and external IP same value as default
		}
		return nil
	})

	fs.Func("i", "name of the internal NIC receiving packet from proxy. Default value is equal to the name of NIC by -e flag.", func(nicName string) error {
		inf, err := net.InterfaceByName(nicName)
		if err != nil {
			return fmt.Errorf("failed to handle '%s' with error: %v", nicName, err)
		}
		conf.InternalIP = GetIPAddrFromInterface(inf)
		if conf.InternalIP == nil {
			return fmt.Errorf("failed to get IPv4 from '%s'", nicName, err)
		}
		return nil
	})

	conf.ProxyIP = net.IPv4(192, 168, 1, 1)
	fs.Func("x", "IP address for the virtual IP for this proxy.", func(ipv4 string) error {
		conf.ProxyIP = net.ParseIP(ipv4)
		if conf.ProxyIP == nil || conf.ProxyIP.To4() == nil {
			return fmt.Errorf("failed to validate '%s' for proxy's IP.", ipv4)
		}
		return nil
	})

	fs.Func("r", "remote IP address allowed to access to this machine.", func(ipv4 string) error {
		conf.RemoteIP = net.ParseIP(ipv4)
		if conf.RemoteIP == nil || conf.RemoteIP.To4() == nil {
			return fmt.Errorf("failed to validate '%s' for remote IP.", ipv4)
		}
		return nil
	})

	if err := fs.Parse(args); err != nil {
		return err
	}

	if err := conf.validate(); err != nil {
		return err
	}

	return nil
}

func main() {
	conf := Config{}
	fs := flag.NewFlagSet("shinobi", flag.ContinueOnError)

	if err := commandInit(fs, &conf, os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		printf(3, "failed to parse command line with error: %v\n", err)
		fs.PrintDefaults()
		os.Exit(1)
	}

	if err := Run(&conf); err != nil {
		printf(3, "failed to run process with error: %v\n", err)
	}
}
