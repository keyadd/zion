package route

import (
	"fmt"
	"log"
	"net"
	"zion.com/zion/utils"
)

var v4Gw string
var v6Gw string
var v4Gateway string
var v6Gateway string
var v6Name string
var serverIPv4 *net.IPAddr
var serverIPv6 *net.IPAddr

func Route(Name string, Dns string, v4gw string, v6gw string, addr string) error {
	_, v4Gateway, _ = utils.GetInternalIPv4()
	v6Name, v6Gateway, _ = utils.GetInternalIPv6()

	IPv4, err := net.ResolveIPAddr("ip4", addr)
	if err != nil {
		log.Println("route net.ResolveIPAddr IPv4", err)
		//return err
	}
	IPv6, err := net.ResolveIPAddr("ip6", addr)
	if err != nil {
		log.Println("route net.ResolveIPAddr IPv6", err)
		//return err
	}
	serverIPv4 = IPv4
	serverIPv6 = IPv6

	log.Println(serverIPv4)
	log.Println(serverIPv6)

	v4Gw = v4gw
	v6Gw = v4gw
	log.Printf("Name %s , Dns %s , v4Gw %s , serverIp %s localGateway %s \n", Name, Dns, v4Gw, serverIPv4, v4Gateway)
	if IPv4 != nil && IPv6 != nil {
		utils.RunCmd("route", "-q", "-n", "add", "-inet", serverIPv4.String(), "-gateway", v4Gateway)
		utils.RunCmd("route", "-q", "-n", "add", "-inet6", serverIPv6.String(), "-gateway", v6Gateway)
		utils.RunCmd("route", "-q", "-n", "delete", "-inet", "0.0.0.0")
		utils.RunCmd("route", "-q", "-n", "delete", "-inet6", "::")
		utils.RunCmd("route", "-q", "-n", "add", "-inet", "default", v4gw)
		utils.RunCmd("route", "-q", "-n", "add", "-inet6", "default", v6gw+"%"+Name)

		utils.RunCmd("killall", "-HUP", "mDNSResponder")
		utils.RunCmd("dscacheutil", "-flushcache")
	} else if IPv4 != nil && IPv6 == nil {
		fmt.Println("ipv4")
		utils.RunCmd("route", "-q", "-n", "add", "-inet", serverIPv4.String(), "-gateway", v4Gateway)
		utils.RunCmd("route", "-q", "-n", "delete", "-inet", "0.0.0.0")
		utils.RunCmd("route", "-q", "-n", "delete", "-inet6", "::")
		utils.RunCmd("route", "-q", "-n", "add", "-inet", "default", v4gw)
		utils.RunCmd("route", "-q", "-n", "add", "-inet6", "default", v6gw+"%"+Name)

		utils.RunCmd("killall", "-HUP", "mDNSResponder")
		utils.RunCmd("dscacheutil", "-flushcache")
	} else if IPv4 == nil && IPv6 != nil {
		utils.RunCmd("route", "-q", "-n", "add", "-inet6", serverIPv6.String(), "-gateway", v6Gateway)
		utils.RunCmd("route", "-q", "-n", "delete", "-inet6", "::")
		utils.RunCmd("route", "add", "default", v6gw+"%"+Name)

		utils.RunCmd("killall", "-HUP", "mDNSResponder")
		utils.RunCmd("dscacheutil", "-flushcache")
	} else {
		fmt.Println("route error")
	}

	return nil
}

func RetractRoute() {
	fmt.Println(serverIPv6)
	fmt.Println(serverIPv4)
	if serverIPv4 != nil && serverIPv6 != nil {
		utils.RunCmd("route", "-q", "-n", "delete", "-inet", serverIPv4.String())
		utils.RunCmd("route", "-q", "-n", "delete", "-inet6", serverIPv6.String())
		utils.RunCmd("route", "-q", "-n", "change", "-inet", "default", v4Gateway)
		utils.RunCmd("route", "-q", "-n", "change", "-inet6", "default", v6Gateway+"%"+v6Name)

		utils.RunCmd("killall", "-HUP", "mDNSResponder")
		utils.RunCmd("dscacheutil", "-flushcache")
	} else if serverIPv4 != nil && serverIPv6 == nil {
		utils.RunCmd("route", "-q", "-n", "delete", "-inet", serverIPv4.String())
		utils.RunCmd("route", "change", "default", v4Gateway)
		utils.RunCmd("route", "-q", "-n", "add", "-inet6", "default", v6Gateway+"%"+v6Name)

		utils.RunCmd("killall", "-HUP", "mDNSResponder")
		utils.RunCmd("dscacheutil", "-flushcache")
	} else if serverIPv4 == nil && serverIPv6 != nil {
		utils.RunCmd("route", "-q", "-n", "delete", "-inet6", serverIPv6.String())
		utils.RunCmd("route", "-q", "-n", "change", "-inet6", "default", v6Gateway+"%"+v6Name)
		utils.RunCmd("route", "-q", "-n", "add", "-inet", "default", v4Gateway)

		utils.RunCmd("killall", "-HUP", "mDNSResponder")
		utils.RunCmd("dscacheutil", "-flushcache")
	} else {
		fmt.Println("route error")
	}

}
