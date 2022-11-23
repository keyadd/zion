package tun

import (
	"errors"
	"fmt"
	"github.com/songgao/water"
	"io"
	"net"
	"zion.com/zion/utils"
)

func OpenLinuxDevice(name, v4Addr, v6Addr string, dnsServers []string) (io.ReadWriteCloser, error) {
	cfg := water.Config{
		DeviceType: water.TUN,
	}
	cfg.Name = name
	tunDev, err := water.New(cfg)
	if err != nil {
		return nil, err
	}
	name = tunDev.Name()

	v4, ipv4Net, err := net.ParseCIDR(v4Addr)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("parse IPv4 addr error: %v", err))
	}

	v6, _, err := net.ParseCIDR(v6Addr)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("parse IPv6 addr error: %v", err))
	}

	if utils.IsIPv4Bool(v4) {
		gw := ipv4Net.IP.To4()
		gw[3]++
		utils.RunCmd("ip", "addr", "add", v4Addr, "dev", name)
		utils.RunCmd("ip", "link", "set", "dev", name, "up")
	}
	if utils.IsIPv6Bool(v6) {
		utils.RunCmd("ip", "-6", "addr", "add", v6Addr, "dev", name)
		utils.RunCmd("ip", "-6", "link", "set", "dev", name, "up")
	} else {
		return nil, errors.New("invalid IP address")
	}

	return tunDev, nil
}
