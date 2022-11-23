package utils

import (
	"crypto/sha256"
	"fmt"
	"github.com/songgao/water/waterutil"
	"golang.org/x/crypto/chacha20poly1305"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
)

func GetIPv4(b []byte) (srcIPv4 string, dstIPv4 string) {
	if waterutil.IPv4Protocol(b) == waterutil.TCP || waterutil.IPv4Protocol(b) == waterutil.UDP || waterutil.IPv4Protocol(b) == waterutil.ICMP {
		srcIp := waterutil.IPv4Source(b)
		dstIp := waterutil.IPv4Destination(b)
		return srcIp.To4().String(), dstIp.To4().String()
	}
	return "", ""
}

func GetIP(b []byte) (src net.IP, dst net.IP) {
	if IsIPv4(b) {
		if waterutil.IPv4Protocol(b) == waterutil.TCP || waterutil.IPv4Protocol(b) == waterutil.UDP || waterutil.IPv4Protocol(b) == waterutil.ICMP {
			src := IPv4Header(b).Src()
			dst := IPv4Header(b).Dst()
			return src, dst
		}
		return nil, nil
	} else if IsIPv6(b) {
		src := IPv6Header(b).Src()
		dst := IPv6Header(b).Dst()
		return src, dst
	} else {
		return nil, nil
	}
}

func GetInternalIPv6() (name string, gateway string, network string) {
	ifaces := getAllPhysicalInterfaces()
	if len(ifaces) == 0 {
		return
	}
	netAddrs, _ := ifaces[0].Addrs()
	for _, addr := range netAddrs {
		ip, ok := addr.(*net.IPNet)
		if !ok {
			fmt.Println("error")
		}
		if ok && ip.IP.To4() == nil && !ip.IP.IsLoopback() {

			_, ipNet, _ := net.ParseCIDR(ip.String())
			to6 := ipNet.IP.To16()
			network = strings.Join([]string{to6.String() + "0", strings.Split(ip.String(), "/")[1]}, "/")
			gateway = to6.String() + "1"
			name = ifaces[0].Name

			break
		}
	}
	return name, gateway, network
}

func GetInternalIPv4() (name string, gateway string, network string) {
	ifaces := getAllPhysicalInterfaces()
	if len(ifaces) == 0 {
		return "", "", ""
	}
	netAddrs, _ := ifaces[0].Addrs()
	for _, addr := range netAddrs {
		ip, ok := addr.(*net.IPNet)
		if ok && ip.IP.To4() != nil && !ip.IP.IsLoopback() {
			ipNet := ip.IP.To4().Mask(ip.IP.DefaultMask()).To4()
			network = strings.Join([]string{ipNet.String(), strings.Split(ip.String(), "/")[1]}, "/")
			ipNet[3]++
			gateway = ipNet.String()
			name = ifaces[0].Name
			//cmd.Gateway = gateway
			//log.Printf("physical interface %v gateway %v network %v", name, gateway, network)
			break
		}
	}
	return name, gateway, network
}

func getAllPhysicalInterfaces() []net.Interface {
	ifaces, err := net.Interfaces()
	if err != nil {
		log.Println(err)
		return nil
	}

	var outInterfaces []net.Interface
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback == 0 && iface.Flags&net.FlagUp == 1 && isPhysicalInterface(iface.Name) {
			netAddrs, _ := iface.Addrs()
			if len(netAddrs) > 0 {
				outInterfaces = append(outInterfaces, iface)
			}
		}
	}
	return outInterfaces
}

func isPhysicalInterface(addr string) bool {
	prefixArray := []string{"ens", "enp", "enx", "eno", "eth", "en0", "wlan", "wlp", "wlo", "wlx", "wifi0", "lan0", "en5", "en6"}
	for _, pref := range prefixArray {
		if strings.HasPrefix(strings.ToLower(addr), pref) {
			return true
		}
	}
	return false
}

// =================== chacha20poly1305 ======================

// EncryptChacha1305 加密
func EncryptChacha1305(origData []byte, key string) (encrypted []byte) {
	newKey := sha256.Sum256([]byte(key))

	aead, _ := chacha20poly1305.New(newKey[:])

	nonce := make([]byte, chacha20poly1305.NonceSize)

	encrypted = aead.Seal(nil, nonce, origData, nil)
	return encrypted
}

// DecryptChacha1305 解密
func DecryptChacha1305(encrypted []byte, key string) (decrypted []byte) {
	newKey := sha256.Sum256([]byte(key))

	aead, _ := chacha20poly1305.New(newKey[:])

	nonce := make([]byte, chacha20poly1305.NonceSize)

	decrypted, _ = aead.Open(nil, nonce, encrypted, nil)
	return decrypted
}

func IsIP(ip string) bool {
	if strings.Contains(ip, "/") {
		netIP, _, err := net.ParseCIDR(ip)
		if err != nil {
			return false
		}
		return netIP.To4() != nil
	}
	netIP := net.ParseIP(ip)
	if netIP == nil {
		return false
	}
	return netIP.To4() != nil
}

func RunCmd(c string, args ...string) {
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

func IsIPv4Bool(ip net.IP) bool {
	if ip.To4() != nil {
		return true
	}
	return false
}

func IsIPv6Bool(ip net.IP) bool {
	// To16() also valid for ipv4, ensure it's not an ipv4 address
	if ip.To4() != nil {
		return false
	}
	if ip.To16() != nil {
		return true
	}
	return false
}
