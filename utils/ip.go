package utils

import (
	"CloudflareSpeedTest/config"
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/RoaringBitmap/roaring"
)

func LoadResultIPV4(resultIPV4File string) *[]uint32 {
	var bestIPV4 = make([]uint32, 0)
	data, err := os.ReadFile(resultIPV4File)
	if err != nil {
		return &bestIPV4
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		ipv4Str := line[0:strings.Index(line, ",")]
		v4, err := IPStringToUint32(ipv4Str)
		if err != nil {
			return &bestIPV4
		}
		bestIPV4 = append(bestIPV4, v4)
	}
	return &bestIPV4
}

func NetIPAddrIPV4toUint32(ip *net.IPAddr) uint32 {
	// 取 IP 字段，并转成 4 字节格式
	v4 := ip.IP.To4()
	if v4 == nil {
		panic("not an IPv4 address")
	}
	// v4[0] 是最高位，按大端解析
	return binary.BigEndian.Uint32(v4)
}

func NetIPIPV4toUint32(ip *net.IP) uint32 {
	// 取 IP 字段，并转成 4 字节格式
	v4 := ip.To4()
	if v4 == nil {
		panic("not an IPv4 address")
	}
	// v4[0] 是最高位，按大端解析
	return binary.BigEndian.Uint32(v4)
}

func Uint32toNetIPAddrIPV4(ip uint32) *net.IPAddr {
	v4 := make(net.IP, 4)
	binary.BigEndian.PutUint32(v4, ip)
	return &net.IPAddr{IP: v4}
}

func IPStringToUint32(ip string) (uint32, error) {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return 0, fmt.Errorf("invalid IP address: %s", ip)
	}
	v4 := parsedIP.To4()
	if v4 == nil {
		return 0, fmt.Errorf("not an IPv4 address: %s", ip)
	}
	return binary.BigEndian.Uint32(v4), nil
}

func loadIPV4RB(dbFile string) *roaring.Bitmap {
	var rb = roaring.New()
	data, err := os.ReadFile(dbFile)
	if err != nil {
		return rb
	}
	_, err = rb.ReadFrom(bytes.NewBuffer(data))
	if err != nil {
		return rb
	}
	return rb
}

func saveIPV4RB(rb *roaring.Bitmap, dbFile string) error {
	buf := new(bytes.Buffer)
	_, err := rb.WriteTo(buf)
	if err != nil {
		return err
	}
	err = os.WriteFile(dbFile, buf.Bytes(), 0644)
	return err
}

func SaveBestAllowDenyIPV4(
	allowIPV4 *[]uint32,
	denyIPV4 *[]uint32,
	allowIPV4RBFile string,
	denyIPV4RBFile string) error {
	allowIPV4RB := loadIPV4RB(allowIPV4RBFile)
	denyIPV4RB := loadIPV4RB(denyIPV4RBFile)
	for _, ip := range *allowIPV4 {
		allowIPV4RB.Add(ip)
	}
	for _, ip := range *denyIPV4 {
		denyIPV4RB.Add(ip)
	}
	err := saveIPV4RB(allowIPV4RB, allowIPV4RBFile)
	if err != nil {
		return err
	}
	err = saveIPV4RB(denyIPV4RB, denyIPV4RBFile)
	if err != nil {
		return err
	}
	return nil
}

func IsIPv4(ip string) bool {
	return strings.Contains(ip, ".")
}

// func isIPv4(ip string) bool {
// 	parsedIP := net.ParseIP(ip)
// 	return parsedIP != nil && parsedIP.To4() != nil
// }

func IsIPV4CIDR(cidr string) bool {
	return strings.Contains(cidr, ".") && strings.Contains(cidr, "/")
}

func IsIPV6CIDR(cidr string) bool {
	return strings.Contains(cidr, ":") && strings.Contains(cidr, "/")
}

func loadCIDRTextSlice(file string) ([]string, error) {
	var cidrs []string
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}

	// 按行读取，去除空行和注释
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cidrs = append(cidrs, line)
	}

	return cidrs, nil
}

func getIPsByCIDRs(
	cidrs []string,
	want int,
	bestIPV4 *[]uint32,
	allowIPV4RB *roaring.Bitmap,
	denyIPV4RB *roaring.Bitmap) ([]*net.IPAddr, error) {
	var ips []*net.IPAddr
	var ipsRB roaring.Bitmap
	// get ip from unvisited ip and allow ip
	testAllowIPV4Num := uint32(float32(want) * config.TestAllowIPV4NumRatio)
	testAllowUPV4PerCIDRNum := int(float32(testAllowIPV4Num) / float32((len(cidrs))))
	if testAllowUPV4PerCIDRNum == 0 {
		testAllowUPV4PerCIDRNum = 1
	}
	for _, cidr := range cidrs {
		// fmt.Printf("get ip from cidr: %s\n", cidr)
		ip, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, err
		}
		// 只支持 IPv4
		ones, bits := ipnet.Mask.Size()
		if bits != 32 {
			return nil, fmt.Errorf("only ipv4 supported")
		}
		total := 1 << (bits - ones) // 2^(32-ones)

		// 基础地址转 uint32
		base := NetIPIPV4toUint32(&ip)
		end := base + uint32(total)
		cidrAllowIPV4 := make([]uint32, 0)
		for i := base; i < end && want > 0; i++ {
			if denyIPV4RB.Contains(i) {
				continue
			}
			if allowIPV4RB.Contains(i) {
				cidrAllowIPV4 = append(cidrAllowIPV4, i)
			}
			ipsRB.Add(i)
			ips = append(ips, Uint32toNetIPAddrIPV4(i))
			want--
		}
		// random choose testAllowUPV4PerCIDRNum ip from base to lastVisited
		config.Rand.Shuffle(len(cidrAllowIPV4), func(i, j int) {
			cidrAllowIPV4[i], cidrAllowIPV4[j] = cidrAllowIPV4[j], cidrAllowIPV4[i]
		})
		for i := 0; i < testAllowUPV4PerCIDRNum && i < len(cidrAllowIPV4); i++ {
			ipsRB.Add(cidrAllowIPV4[i])
			ips = append(ips, Uint32toNetIPAddrIPV4(cidrAllowIPV4[i]))
		}
	}
	// get ip from result ip
	for _, ip := range *bestIPV4 {
		if ipsRB.Contains(ip) {
			continue
		}
		ips = append(ips, Uint32toNetIPAddrIPV4(ip))
	}
	return ips, nil
}

func GetIPs(
	cidrFile string,
	want int,
	lastOutputFile string,
	allowIPV4RBFile string,
	denyIPV4RBFile string) []*net.IPAddr {
	cidrs, err := loadCIDRTextSlice(cidrFile)
	if err != nil {
		return nil
	}
	resultIPV4 := LoadResultIPV4(lastOutputFile)
	allowIPV4RB := loadIPV4RB(allowIPV4RBFile)
	denyIPV4RB := loadIPV4RB(denyIPV4RBFile)
	ips, err := getIPsByCIDRs(cidrs, want, resultIPV4, allowIPV4RB, denyIPV4RB)
	if err != nil {
		return nil
	}
	return ips
}
