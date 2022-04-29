package route

import (
	"log"
	"net"
	"os"
	"os/exec"
	"zion.com/zion/utils"
)

var Gateway string
var serverIP string

func Route(Name string, Dns string, v4Gw string, addr string) error {
	physicalIface, localGateway, _ := utils.GetPhysicalInterface()
	//fmt.Println(localGateway)
	Gateway = localGateway
	ip, err := net.ResolveIPAddr("ip", addr)
	if err != nil {
		log.Println("route net.ResolveIPAddr", err)
		return err
	}
	serverIP = ip.String()
	log.Println(serverIP)
	log.Printf("Name %s , Dns %s , v4Gw %s , serverIp %s localGateway %s \n", Name, Dns, v4Gw, serverIP, localGateway)
	if physicalIface != "" {
		if utils.IsIP(serverIP) {
			execCmd("route", "-q", "-n", "add", "-inet", serverIP, "-gateway", localGateway)
			execCmd("route", "delete", "0.0.0.0")
			execCmd("route", "add", "default", v4Gw)
			//execCmd("route", "add", "-inet6", "default", "fd42:42:42::2")
			execCmd("killall", "-HUP", "mDNSResponder")
			execCmd("dscacheutil", "-flushcache")
		} else {
			execCmd("route", "-q", "-n", "add", "-inet6", serverIP, "-gateway", localGateway)
			execCmd("route", "delete", "0.0.0.0")
			execCmd("route", "add", "default", v4Gw)
			//execCmd("route", "add", "-inet6", "default", "fd42:42:42::2")
			execCmd("killall", "-HUP", "mDNSResponder")
			execCmd("dscacheutil", "-flushcache")
		}

		//execCmd("route", "add", "0.0.0.0/1", "-interface", tunName)
		//execCmd("route", "add", "128.0.0.0/1", "-interface", tunName)
		//execCmd("route", "add", "-host", "1.1.1.1", "dev", tunName)
		//execCmd("route", "add", "13.251.188.177", "-interface", tunName)
		//execCmd("route", "delete", "default")
		//execCmd("route", "delete", "-inet6", "default")

		//execCmd("route", "change", "default", tunGw)
	}
	return nil
}

func RetractRoute() {
	//fmt.Println(Gateway)
	if utils.IsIP(serverIP) {
		execCmd("route", "-q", "-n", "delete", "-inet", serverIP)
		execCmd("route", "delete", "0.0.0.0")
		execCmd("route", "add", "default", Gateway)
		execCmd("killall", "-HUP", "mDNSResponder")
		execCmd("dscacheutil", "-flushcache")
	} else {
		execCmd("route", "-q", "-n", "delete", "-inet6", serverIP)
		execCmd("route", "delete", "0.0.0.0")
		execCmd("route", "add", "default", Gateway)
		execCmd("killall", "-HUP", "mDNSResponder")
		execCmd("dscacheutil", "-flushcache")
	}

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
