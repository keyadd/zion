package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"github.com/songgao/water/waterutil"
	"log"
	"net"
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

func GetPhysicalInterface() (name string, gateway string, network string) {
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
	prefixArray := []string{"ens", "enp", "enx", "eno", "eth", "en0", "wlan", "wlp", "wlo", "wlx", "wifi0", "lan0"}
	for _, pref := range prefixArray {
		if strings.HasPrefix(strings.ToLower(addr), pref) {
			return true
		}
	}
	return false
}

const (
	sKey        = "1234567890000000"
	ivParameter = "dde4b1f8a9e6b814"
)

var Data string

//加密
func PswEncrypt(src []byte) []byte {
	key := []byte(sKey)
	iv := []byte(ivParameter)

	result, err := Aes128Encrypt(src, key, iv)
	if err != nil {
		log.Println(err)
		return nil
	}
	return []byte(base64.RawStdEncoding.EncodeToString(result))
}

//解密
func PswDecrypt(src []byte) []byte {

	key := []byte(sKey)
	iv := []byte(ivParameter)

	var result []byte
	var err error

	result, err = base64.RawStdEncoding.DecodeString(string(src))
	if err != nil {
		log.Println(err)
		return nil
	}
	origData, err := Aes128Decrypt(result, key, iv)
	if err != nil {
		log.Println(err)
		return nil
	}
	return origData

}
func Aes128Encrypt(origData, key []byte, IV []byte) ([]byte, error) {
	if key == nil || len(key) != 16 {
		return nil, nil
	}
	if IV != nil && len(IV) != 16 {
		return nil, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	origData = PKCS5Padding(origData, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, IV[:blockSize])
	crypted := make([]byte, len(origData))
	// 根据CryptBlocks方法的说明，如下方式初始化crypted也可以
	blockMode.CryptBlocks(crypted, origData)
	return crypted, nil
}

func Aes128Decrypt(crypted, key []byte, IV []byte) ([]byte, error) {
	if key == nil || len(key) != 16 {
		return nil, nil
	}
	if IV != nil && len(IV) != 16 {
		return nil, nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, IV[:blockSize])
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = PKCS5UnPadding(origData)
	return origData, nil
}

func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	// 去掉最后一个字节 unpadding 次
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}
