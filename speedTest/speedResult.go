package speedTest

import (
	"CloudflareSpeedTest/config"
	"encoding/csv"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"time"
)

// 判断文件是否存在
func fileExists(filename string) (bool, error) {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err // 可能是权限问题等其他错误
	}
	return true, nil
}

// 获取文件的修改时间，并判断是否为今天
func isFileModifiedToday(filename string) (bool, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return false, err // 文件不存在或权限问题
	}

	modTime := info.ModTime()
	today := time.Now().Truncate(24 * time.Hour) // 将当前时间截断到当天的开始（00:00:00）
	tomorrow := today.Add(24 * time.Hour)        // 明天的开始时间

	// 如果文件的修改时间在今天和明天之间，则认为它是今天的
	return modTime.After(today) && modTime.Before(tomorrow), nil
}

type SpeedResult struct {
	IP            *net.IPAddr
	Sended        int
	Received      int
	Delay         time.Duration
	Colo          string
	LossRate      float32
	DownloadSpeed float64
}

func (s *SpeedResult) getLossRate() float32 {
	if s.Sended <= 0 {
		return 1.0
	}
	if s.LossRate == 0 {
		pingLost := s.Sended - s.Received
		s.LossRate = float32(pingLost) / float32(s.Sended)
	}
	return s.LossRate
}

func (s *SpeedResult) toStringSlice() []string {
	result := make([]string, 7)
	result[0] = s.IP.String()
	result[1] = strconv.Itoa(s.Sended)
	result[2] = strconv.Itoa(s.Received)
	result[3] = strconv.FormatFloat(float64(s.getLossRate()), 'f', 2, 32)
	result[4] = strconv.FormatFloat(s.Delay.Seconds()*1000, 'f', 2, 32)
	result[5] = strconv.FormatFloat(s.DownloadSpeed/1024/1024, 'f', 2, 32)
	result[6] = s.Colo
	if result[6] == "" {
		result[6] = "N/A"
	}
	return result
}

func (s *SpeedResult) fromStringSlice(data []string) error {
	if len(data) != 7 {
		return fmt.Errorf("数据格式错误")
	}
	s.IP, _ = net.ResolveIPAddr("ip", data[0])
	s.Sended, _ = strconv.Atoi(data[1])
	s.Received, _ = strconv.Atoi(data[2])
	_lossRate, _ := strconv.ParseFloat(data[3], 64)
	s.LossRate = float32(_lossRate)
	s.Delay, _ = time.ParseDuration(data[4] + "ms")
	s.DownloadSpeed, _ = strconv.ParseFloat(data[5], 64)
	s.Colo = data[6]
	return nil
}

type SpeedResultSlice []SpeedResult

func NewSpeedResultSlice(ips []*net.IPAddr) *SpeedResultSlice {
	s := new(SpeedResultSlice)
	for i := 0; i < len(ips); i++ {
		*s = append(*s, SpeedResult{
			IP:            ips[i],
			Sended:        0,
			Received:      0,
			Delay:         0,
			Colo:          "",
			LossRate:      0,
			DownloadSpeed: 0,
		})
	}
	return s
}

func (s *SpeedResultSlice) toStringSlice() [][]string {
	var result [][]string
	for i := 0; i < len(*s); i++ {
		result = append(result, (*s)[i].toStringSlice())
	}
	return result
}

func (s *SpeedResultSlice) fromStringSlice(data [][]string) error {
	if len(data) == 0 {
		return fmt.Errorf("数据格式错误")
	}
	for i := 0; i < len(data); i++ {
		speedResult := SpeedResult{}
		err := speedResult.fromStringSlice(data[i])
		if err != nil {
			return err
		}
		*s = append(*s, speedResult)
	}
	return nil
}

func (s *SpeedResultSlice) SaveSpeedResultSlice(outputFile string, num int) {
	if len(*s) == 0 {
		return
	}
	fp, err := os.Create(outputFile)
	if err != nil {
		fmt.Printf("创建文件[%s]失败：%v", outputFile, err)
		return
	}
	defer fp.Close()
	ws := NewSpeedResultSlice(nil)
	for i := 0; i < len(*s) && i < num; i++ {
		if (*s)[i].getLossRate() == 1.0 || (*s)[i].Delay == config.MaxDelay {
			continue
		}
		*ws = append(*ws, (*s)[i])
	}
	var lines [][]string
	lines = append(lines, []string{"IP 地址", "已发送", "已接收", "丢包率", "平均延迟", "下载速度(MB/s)", "地区码"})
	lines = append(lines, ws.toStringSlice()...)
	w := csv.NewWriter(fp) //创建一个新的写入文件流
	_ = w.WriteAll(lines)
	w.Flush()
}

func (s *SpeedResultSlice) LoadSpeedResultSlice(inputFile string) error {
	fp, err := os.Open(inputFile)
	if err != nil {
		return err
	}
	defer fp.Close()
	r := csv.NewReader(fp)
	lines, err := r.ReadAll()
	if err != nil {
		return err
	}
	err = s.fromStringSlice(lines[1:])
	return err
}

// func (s *SpeedResultSlice) SortByDelay() {
// 	sort.Slice(*s, func(i, j int) bool {
// 		return (*s)[i].Delay < (*s)[j].Delay
// 	})
// }
// func (s *SpeedResultSlice) SortByLossRate() {
// 	sort.Slice(*s, func(i, j int) bool {
// 		return (*s)[i].getLossRate() < (*s)[j].getLossRate()
// 	})
// }
// func (s *SpeedResultSlice) SortByDownloadSpeed() {
// 	sort.Slice(*s, func(i, j int) bool {
// 		return (*s)[i].DownloadSpeed < (*s)[j].DownloadSpeed
// 	})
// }

func (s *SpeedResultSlice) SortByDownloadSpeedDelayLossRate() {
	sort.Slice(*s, func(i, j int) bool {
		if (*s)[i].DownloadSpeed == (*s)[j].DownloadSpeed {
			if (*s)[i].Delay == (*s)[j].Delay {
				return (*s)[i].getLossRate() < (*s)[j].getLossRate()
			}
			return (*s)[i].Delay < (*s)[j].Delay
		}
		return (*s)[i].DownloadSpeed > (*s)[j].DownloadSpeed
	})
}

func (s *SpeedResultSlice) Print(num int) {
	if num <= 0 {
		return
	}
	if len(*s) <= 0 { // IP数组长度(IP数量) 大于 0 时继续
		fmt.Println("\n[信息] 完整测速结果 IP 数量为 0, 跳过输出结果。")
		return
	}
	if len(*s) < num { // 如果IP数组长度(IP数量) 小于  打印次数，则次数改为IP数量
		num = len(*s)
	}
	// fmt.Printf("\033[31m slice and first item address %p %p \033[0m\n", s, &(*s)[0])
	dateString := make([][]string, 0)
	for i := 0; i < num; i++ {
		dateString = append(dateString, (*s)[i].toStringSlice())
	}
	headFormat := "\033[34m%-16s%-5s%-5s%-5s%-6s%-12s%-5s\033[0m\n"
	dataFormat := "%-18s%-8s%-8s%-8s%-10s%-16s%-8s\n"
	hasIPV6 := false
	for i := 0; i < num; i++ { // 如果要输出的 IP 中包含 IPv6，那么就需要调整一下间隔
		if hasIPV6 {
			headFormat = "\033[34m%-40s%-5s%-5s%-5s%-6s%-12s%-5s\033[0m\n"
			dataFormat = "%-42s%-8s%-8s%-8s%-10s%-16s%-8s\n"
			break
		}
	}
	fmt.Printf(headFormat, "IP 地址", "已发送", "已接收", "丢包率", "平均延迟", "下载速度(MB/s)", "地区码")
	for i := 0; i < num; i++ {
		fmt.Printf(dataFormat, dateString[i][0], dateString[i][1], dateString[i][2], dateString[i][3], dateString[i][4], dateString[i][5], dateString[i][6])
	}
}
