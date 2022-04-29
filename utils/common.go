package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"github.com/songgao/water/waterutil"
	"golang.org/x/crypto/chacha20poly1305"
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

func GetIP(b []byte) (src string, dst string) {

	if IsIPv4(b) {
		if waterutil.IPv4Protocol(b) == waterutil.TCP || waterutil.IPv4Protocol(b) == waterutil.UDP || waterutil.IPv4Protocol(b) == waterutil.ICMP {
			src := IPv4Header(b).Src().String()
			dst := IPv4Header(b).Dst().String()
			return src, dst
		}
		return "", ""
	} else if IsIPv6(b) {
		src := IPv6Header(b).Src().String()
		dst := IPv6Header(b).Dst().String()
		return src, dst
	} else {
		return "", ""
	}
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

// =================== ECB ======================

// AesEncryptECB 加密
func AesEncryptECB(origData []byte, key []byte) (encrypted []byte) {
	cipher, _ := aes.NewCipher(generateKey(key))
	length := (len(origData) + aes.BlockSize) / aes.BlockSize
	plain := make([]byte, length*aes.BlockSize)
	copy(plain, origData)
	pad := byte(len(plain) - len(origData))
	for i := len(origData); i < len(plain); i++ {
		plain[i] = pad
	}
	encrypted = make([]byte, len(plain))
	// 分组分块加密
	for bs, be := 0, cipher.BlockSize(); bs <= len(origData); bs, be = bs+cipher.BlockSize(), be+cipher.BlockSize() {
		cipher.Encrypt(encrypted[bs:be], plain[bs:be])
	}

	return encrypted
}

// AesDecryptECB 解密
func AesDecryptECB(encrypted []byte, key []byte) (decrypted []byte) {
	cipher, _ := aes.NewCipher(generateKey(key))
	decrypted = make([]byte, len(encrypted))
	//
	for bs, be := 0, cipher.BlockSize(); bs < len(encrypted); bs, be = bs+cipher.BlockSize(), be+cipher.BlockSize() {
		cipher.Decrypt(decrypted[bs:be], encrypted[bs:be])
	}

	trim := 0
	if len(decrypted) > 0 {
		trim = len(decrypted) - int(decrypted[len(decrypted)-1])
	}

	return decrypted[:trim]
}

func generateKey(key []byte) (genKey []byte) {
	genKey = make([]byte, 16)
	copy(genKey, key)
	for i := 16; i < len(key); {
		for j := 0; j < 16 && i < len(key); j, i = j+1, i+1 {
			genKey[j] ^= key[i]
		}
	}
	return genKey
}

// =================== CBC ======================

const (
	sKey        = "1234567890000000"
	ivParameter = "dde4b1f8a9e6b814"
)

var Data string

// PswEncrypt 加密
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

// PswDecrypt 解密
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
