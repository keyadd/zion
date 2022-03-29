package route

import (
	"log"
	"net"
	"os"
	"os/exec"
	"zion.com/zion/utils"
)

var Gateway string

func Route(tunName string, tunDns string, tunGw string, addr string, c chan os.Signal) {
	physicalIface, localGateway, _ := utils.GetPhysicalInterface()
	//fmt.Println(localGateway)
	Gateway = localGateway
	ip, err := net.ResolveIPAddr("ip", addr)
	if err != nil {
		log.Println("route net.ResolveIPAddr", err)
	}
	serverIP := ip.String()
	log.Println(physicalIface)
	//log.Printf("tunName %s , tunDns %s , tunGw %s , serverIp %s localGateway %s \n", tunName, tunDns, tunGw, serverIP, localGateway)
	if physicalIface != "" {
		execCmd("route", "add", serverIP, localGateway)
		execCmd("route", "add", tunDns, localGateway)
		execCmd("route", "add", "0.0.0.0/1", "-interface", tunName)
		execCmd("route", "add", "128.0.0.0/1", "-interface", tunName)
		execCmd("route", "add", "13.251.188.177", "-interface", tunName)
		execCmd("route", "add", "default", tunGw)
		execCmd("route", "change", "default", tunGw)
	}
}

func RetractRoute() {
	//fmt.Println(Gateway)
	execCmd("route", "add", "default", Gateway)
	execCmd("route", "change", "default", Gateway)
}

func execCmd(c string, args ...string) {
	//log.Printf("exec cmd: %v %v:", c, args)
	cmd := exec.Command(c, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err != nil {
		log.Println("failed to exec cmd:", err)
	}
}