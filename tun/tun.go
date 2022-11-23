package tun

import (
	"io"
	"log"
	"runtime"
)

// OpenDevice 判断不同点系统 根据操作系统 开启虚拟网卡tun 方法
func OpenDevice(name, v4Addr, v6Addr string, dnsServers []string) (io.ReadWriteCloser, error) {
	os := runtime.GOOS
	if os == "linux" {
		return OpenLinuxDevice(name, v4Addr, v6Addr, dnsServers)
	} else if os == "darwin" {
		return OpenDarwinDevice(name, v4Addr, v6Addr, dnsServers)
	} else {
		log.Printf("not support os:%v", os)
	}
	return nil, nil

}
