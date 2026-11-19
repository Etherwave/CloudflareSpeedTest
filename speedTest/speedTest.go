package speedTest

import (
	"CloudflareSpeedTest/config"
	"CloudflareSpeedTest/utils"
	"time"
)

func (s *SpeedResult) SpeedTest(
	tcpPort int,
	tcpConnectTimes int,
	tcpConnectTimeout time.Duration,
	downloadIPTestTimes int,
	downloadTimeout time.Duration,
	downloadURL string,
	downloadTCPPort int,
	bar *utils.Bar) {
	defer bar.Grow(1, "")
	s.TcpTest(tcpPort, tcpConnectTimes, tcpConnectTimeout, nil)
	if s.Received != tcpConnectTimes {
		if config.Debug { // 调试模式下，输出更多信息
			utils.Red.Printf("[调试] IP: %s, TCP 测速失败\n", s.IP.String())
		}
		return
	}
	s.DownloadTest(downloadIPTestTimes, downloadTimeout, downloadURL, downloadTCPPort, nil)
}

func (s *SpeedResultSlice) SpeedTest(
	routines int,
	tcpPort int,
	tcpConnectTimes int,
	tcpConnectTimeout time.Duration,
	downloadIPTestTimes int,
	downloadTimeout time.Duration,
	downloadURL string,
	downloadTCPPort int) {
	workerPool := utils.NewWorkerPool(routines)
	bar := utils.NewBar(len(*s), "", "")
	sc := NewStopCounter(10)
	for i := 0; i < len(*s); i++ {
		sr := &((*s)[i])
		workerPool.Submit(func() {
			if sc.Stopped() {
				return
			}
			sr.SpeedTest(tcpPort, tcpConnectTimes, tcpConnectTimeout, downloadIPTestTimes, downloadTimeout, downloadURL, downloadTCPPort, bar)
			if sr.DownloadSpeed > 0 {
				if sr.DownloadSpeed > 1024*1024 {
					sc.Inc()
				}
				if config.Debug {
					utils.Green.Printf("[调试] IP: %s, 下载速度: %.2f MB/s, 延迟: %s, 丢包率: %.2f%%\n", sr.IP.String(), sr.DownloadSpeed/1024/1024, sr.Delay, sr.getLossRate())
				}
			}
		})
	}
	workerPool.Wait()
	bar.Done()
	workerPool.Stop()
}
