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
	// skip header
	for i := 1; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
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
		if denyIPV4RB.Contains(ip) {
			denyIPV4RB.Remove(ip)
		}
	}
	for _, ip := range *denyIPV4 {
		denyIPV4RB.Add(ip)
		if allowIPV4RB.Contains(ip) {
			allowIPV4RB.Remove(ip)
		}
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

// SelectRandomIPs 从 Bitmap 中随机选择 n 个不重复的 IP (uint32)
func SelectRandomIPs(bm *roaring.Bitmap, n uint32) []uint32 {
	if bm.GetCardinality() == 0 || n <= 0 {
		return nil
	}

	// 1. 将 Bitmap 全部导出为 []uint32 (这一步非常快)
	allIPs := bm.ToArray()

	// 如果 n 大于总数量，返回全部并打乱，或者返回全部
	if n >= uint32(len(allIPs)) {
		// 打乱后返回全部
		config.Rand.Shuffle(len(allIPs), func(i, j int) { allIPs[i], allIPs[j] = allIPs[j], allIPs[i] })
		return allIPs
	}

	// 2. 随机采样：使用 Fisher-Yates 算法只打乱前 n 个位置
	for i := 0; i < int(n); i++ {
		// 在 [i, len-1] 范围内随机选一个下标
		j := config.Rand.IntN(len(allIPs)-i) + i
		allIPs[i], allIPs[j] = allIPs[j], allIPs[i]
	}

	// 返回前 n 个
	return allIPs[:n]
}

func getIPsByCIDRs(
	cidrs []string,
	want uint32,
	bestIPV4 *[]uint32,
	allowIPV4RB *roaring.Bitmap,
	denyIPV4RB *roaring.Bitmap) ([]*net.IPAddr, error) {
	var ips []*net.IPAddr
	var ipsRB roaring.Bitmap
	// get ip from unvisited ip and allow ip
	testUnvisitedIPV4Num := uint32(float32(want) * config.TestUnvisitedIPV4NumRatio)
	LogInfo("testUnvisitedIPV4Num: %d", testUnvisitedIPV4Num)
	var collectIPCnt uint32 = 0
	for _, cidr := range cidrs {
		LogInfo("get ip from cidr: %s", cidr)
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
		for i := base; i < end; i++ {
			if denyIPV4RB.Contains(i) {
				continue
			}
			if allowIPV4RB.Contains(i) {
				continue
			}
			ipsRB.Add(i)
			ips = append(ips, Uint32toNetIPAddrIPV4(i))
			collectIPCnt++
			if collectIPCnt >= testUnvisitedIPV4Num {
				break
			}
		}
		if collectIPCnt >= testUnvisitedIPV4Num {
			break
		}
	}
	// get ip from allowed ip
	testAllowIPV4Num := want - collectIPCnt
	LogInfo("testAllowIPV4Num: %d", testAllowIPV4Num)
	allowedIps := SelectRandomIPs(allowIPV4RB, testAllowIPV4Num)
	LogInfo("allowedIps size: %v", len(allowedIps))
	for _, ip := range allowedIps {
		ipsRB.Add(ip)
		ips = append(ips, Uint32toNetIPAddrIPV4(ip))
	}
	// get ip from result ip
	for _, ip := range *bestIPV4 {
		if ipsRB.Contains(ip) {
			continue
		}
		ips = append(ips, Uint32toNetIPAddrIPV4(ip))
	}
	LogInfo("ips size: %v", len(ips))
	return ips, nil
}

func GetIPs(
	cidrFile string,
	want uint32,
	lastOutputFile string,
	allowIPV4RBFile string,
	denyIPV4RBFile string) []*net.IPAddr {
	cidrs, err := loadCIDRTextSlice(cidrFile)
	if err != nil {
		return nil
	}
	resultIPV4 := LoadResultIPV4(lastOutputFile)
	LogInfo("resultIPV4 size: %v", len(*resultIPV4))
	allowIPV4RB := loadIPV4RB(allowIPV4RBFile)
	LogInfo("allowIPV4RB size: %v", allowIPV4RB.GetCardinality())
	denyIPV4RB := loadIPV4RB(denyIPV4RBFile)
	LogInfo("denyIPV4RB size: %v", denyIPV4RB.GetCardinality())
	ips, err := getIPsByCIDRs(cidrs, want, resultIPV4, allowIPV4RB, denyIPV4RB)
	if err != nil {
		return nil
	}
	return ips
}

func getIPsNumByCIDRs(cidrs []string) (int, error) {
	var totalIpsNum int
	for _, cidr := range cidrs {
		// fmt.Printf("get ip from cidr: %s\n", cidr)
		_, ipnet, err := net.ParseCIDR(cidr)
		if err != nil {
			return 0, err
		}
		// 只支持 IPv4
		ones, bits := ipnet.Mask.Size()
		if bits != 32 {
			return 0, fmt.Errorf("only ipv4 supported")
		}
		total := 1 << (bits - ones) // 2^(32-ones)
		totalIpsNum += total
	}
	return totalIpsNum, nil
}

func ShowIPStatus(
	cidrFile string,
	allowIPV4RBFile string,
	denyIPV4RBFile string) {
	cidrs, _ := loadCIDRTextSlice(cidrFile)
	totalIpsNum, _ := getIPsNumByCIDRs(cidrs)
	allowIPV4RB := loadIPV4RB(allowIPV4RBFile)
	denyIPV4RB := loadIPV4RB(denyIPV4RBFile)

	fmt.Printf("total ipv4 num: %d\n", totalIpsNum)
	fmt.Printf("allow ipv4 num: %d\n", allowIPV4RB.GetCardinality())
	fmt.Printf("deny ipv4 num: %d\n", denyIPV4RB.GetCardinality())
}
