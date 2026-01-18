package speedTest

import (
	"CloudflareSpeedTest/config"
	"CloudflareSpeedTest/utils"
	"fmt"
	"net"
	"time"
)

func (s *SpeedResult) TcpTest(
	tcpPort int,
	tcpConnectTimes int,
	tcpConnectTimeout time.Duration,
	bar *utils.Bar) {
	defer bar.Grow(1, "")
	s.Sended = tcpConnectTimes
	s.Received = 0
	s.Delay = config.MaxDelay
	var totalDelay time.Duration
	for i := 0; i < tcpConnectTimes; i++ {
		startTime := time.Now()
		var fullAddress string
		if utils.IsIPv4(s.IP.String()) {
			fullAddress = fmt.Sprintf("%s:%d", s.IP.String(), tcpPort)
		} else {
			fullAddress = fmt.Sprintf("[%s]:%d", s.IP.String(), tcpPort)
		}
		conn, err := net.DialTimeout("tcp", fullAddress, tcpConnectTimeout)
		if err != nil {
			continue
		}
		defer conn.Close()
		s.Received++
		totalDelay += time.Since(startTime)
	}
	if s.Received == 0 {
		s.Delay = config.MaxDelay
	} else {
		s.Delay = totalDelay / time.Duration(s.Received)
	}
}

func (s *SpeedResultSlice) TcpTest(routines int, tcpPort int, tcpConnectTimes int, tcpConnectTimeout time.Duration) {
	workerPool := utils.NewWorkerPool(routines)
	bar := utils.NewBar(len(*s), "", "")
	for i := 0; i < len(*s); i++ {
		sr := &((*s)[i])
		workerPool.Submit(func() {
			sr.TcpTest(tcpPort, tcpConnectTimes, tcpConnectTimeout, bar)
		})
	}
	workerPool.Wait()
	bar.Done()
	workerPool.Stop()
}
