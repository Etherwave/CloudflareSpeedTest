package main

import (
	"CloudflareSpeedTest/config"
	"CloudflareSpeedTest/speedTest"
	"CloudflareSpeedTest/utils"
	"fmt"
)

func SpeedTest() (s *speedTest.SpeedResultSlice) {
	ips := utils.GetIPs(
		config.Config.CIDRIPV4File,
		config.Config.TestIPNum,
		config.Config.OutputFile,
		config.Config.AllowIPV4RBFile,
		config.Config.DenyIPV4RBFile,
	)
	s = speedTest.NewSpeedResultSlice(ips)
	fmt.Printf("TestMode %s\n", config.Config.TestMode)
	switch config.Config.TestMode {
	case "tcp":
		s.TcpTest(
			config.Config.TcpRoutines,
			config.Config.TcpPort,
			config.Config.TcpConnectTimes,
			config.Config.TcpConnectTimeout,
		)
	case "http":
		s.HttpTest(
			config.Config.HttpColo,
			config.Config.HttpColoSet,
			config.Config.HttpConnectTimes,
			config.Config.HttpConnectTimeout,
			config.Config.HttpRoutines,
			config.Config.HttpStatusCode,
			config.Config.HttpURL,
			config.Config.HttpTCPPort,
		)
	}
	// update ip download speed by last result
	lastSpeedResultSlice := speedTest.NewSpeedResultSlice(nil)
	lastSpeedResultSlice.LoadSpeedResultSlice(config.Config.OutputFile)
	ss := speedTest.SpeedResultSet{}
	for i := 0; i < len(*lastSpeedResultSlice); i++ {
		ss.Add(&(*lastSpeedResultSlice)[i])
	}
	for i := 0; i < len(*s); i++ {
		ssIp := ss.Get((*s)[i].IP.String())
		if ssIp != nil {
			(*s)[i].DownloadSpeed = ssIp.DownloadSpeed
		}
	}
	s.SortByDelayLossRate()
	// 开始下载测速
	if config.Config.EnableDownLoadTest {
		fmt.Printf("Start DownloadTest %s\n", config.Config.DownloadURL)
		s.DownloadTest(
			config.Config.DownloadTestIPNum,
			config.Config.DownloadIPTestTimes,
			config.Config.DownloadTimeout,
			config.Config.DownloadURL,
			config.Config.DownloadTCPPort,
		)
	}
	s.SortByDownloadSpeedDelayLossRate()
	return s
}

func outputResultAllowDenayIPV4(s *speedTest.SpeedResultSlice) error {
	allowIPV4 := []uint32{}
	denyIPV4 := []uint32{}
	for i := 0; i < len(*s); i++ {
		si := (*s)[i]
		isAllow := si.Delay < config.MaxAllowDelay
		ipUint32 := utils.NetIPAddrIPV4toUint32(si.IP)
		if isAllow {
			allowIPV4 = append(allowIPV4, ipUint32)
		} else {
			denyIPV4 = append(denyIPV4, ipUint32)
		}
	}
	err := utils.SaveBestAllowDenyIPV4(
		&allowIPV4,
		&denyIPV4,
		config.Config.AllowIPV4RBFile,
		config.Config.DenyIPV4RBFile,
	)
	if err != nil {
		return err
	}
	s.SaveSpeedResultSlice(config.Config.OutputFile, config.Config.SaveIPNum)
	return nil
}

func updateWebHosts(s *speedTest.SpeedResultSlice) error {
	bestIp := (*s)[0].IP.String()
	err := utils.UpdateHosts(bestIp, config.Config.WebHosts) // 更新hosts文件
	return err
}

func main() {
	fmt.Printf("# etherwave/CloudflareSpeedTest %s \n\n", config.Version)
	err := config.Init()
	if err != nil {
		fmt.Println(err)
		return
	}
	s := SpeedTest() // 获取下载测速结果
	fmt.Println("SpeedTest Done")
	// s.Print(config.Config.TestIPNum)
	s.Print(10)
	err = outputResultAllowDenayIPV4(s)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = updateWebHosts(s)
	if err != nil {
		fmt.Println(err)
		return
	}
}
