package speedTest

import (
	"CloudflareSpeedTest/config"
	"CloudflareSpeedTest/utils"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const (
	bufferSize = 1024
)

func getDialContext(ip *net.IPAddr, tcpPort int) func(ctx context.Context, network, address string) (net.Conn, error) {
	var fakeSourceAddr string
	if utils.IsIPv4(ip.String()) {
		fakeSourceAddr = fmt.Sprintf("%s:%d", ip.String(), tcpPort)
	} else {
		fakeSourceAddr = fmt.Sprintf("[%s]:%d", ip.String(), tcpPort)
	}
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, fakeSourceAddr)
	}
}

// 统一的请求报错调试输出
func printDownloadDebugInfo(ip *net.IPAddr, err error, statusCode int, url, lastRedirectURL string, response *http.Response) {
	finalURL := url // 默认的最终 URL，这样当 response 为空时也能输出
	if lastRedirectURL != "" {
		finalURL = lastRedirectURL // 如果 lastRedirectURL 不是空，说明重定向过，优先输出最后一次要重定向至的目标
	} else if response != nil && response.Request != nil && response.Request.URL != nil {
		finalURL = response.Request.URL.String() // 如果 response 不为 nil，且 Request 和 URL 都不为 nil，则获取最后一次成功的响应地址
	}
	if url != finalURL { // 如果 URL 和最终地址不一致，说明有重定向，是该重定向后的地址引起的错误
		if statusCode > 0 { // 如果状态码大于 0，说明是后续 HTTP 状态码引起的错误
			utils.Red.Printf("[调试] IP: %s, 下载测速终止，HTTP 状态码: %d, 下载测速地址: %s, 出错的重定向后地址: %s\n", ip.String(), statusCode, url, finalURL)
		} else {
			utils.Red.Printf("[调试] IP: %s, 下载测速失败，错误信息: %v, 下载测速地址: %s, 出错的重定向后地址: %s\n", ip.String(), err, url, finalURL)
		}
	} else { // 如果 URL 和最终地址一致，说明没有重定向
		if statusCode > 0 { // 如果状态码大于 0，说明是后续 HTTP 状态码引起的错误
			utils.Red.Printf("[调试] IP: %s, 下载测速终止，HTTP 状态码: %d, 下载测速地址: %s\n", ip.String(), statusCode, url)
		} else {
			utils.Red.Printf("[调试] IP: %s, 下载测速失败，错误信息: %v, 下载测速地址: %s\n", ip.String(), err, url)
		}
	}
}

// return download Speed
func downloadURLByIP(
	downloadTimeout time.Duration,
	downloadURL string,
	ip *net.IPAddr,
	tcpPort int) (float64, string) {
	var lastRedirectURL string // 用于记录最后一次重定向目标，以便在访问错误时输出
	client := &http.Client{
		Transport: &http.Transport{DialContext: getDialContext(ip, tcpPort)},
		Timeout:   downloadTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			lastRedirectURL = req.URL.String() // 记录每次重定向的目标，以便在访问错误时输出
			if len(via) > 10 {                 // 限制最多重定向 10 次
				if config.Debug { // 调试模式下，输出更多信息
					utils.Red.Printf("[调试] IP: %s, 下载测速地址重定向次数过多，终止测速，下载测速地址: %s\n", ip.String(), req.URL.String())
				}
				return http.ErrUseLastResponse
			}
			if req.Header.Get("Referer") == downloadURL { // 当使用默认下载测速地址时，重定向不携带 Referer
				req.Header.Del("Referer")
			}
			return nil
		},
	}
	req, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		if config.Debug { // 调试模式下，输出更多信息
			utils.Red.Printf("[调试] IP: %s, 下载测速请求创建失败，错误信息: %v, 下载测速地址: %s\n", ip.String(), err, downloadURL)
		}
		return 0.0, ""
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/98.0.4758.80 Safari/537.36")

	response, err := client.Do(req)
	timeStart := time.Now() // 开始时间（当前）
	if err != nil {
		if config.Debug { // 调试模式下，输出更多信息
			printDownloadDebugInfo(ip, err, 0, downloadURL, lastRedirectURL, response)
		}
		return 0.0, ""
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		if config.Debug { // 调试模式下，输出更多信息
			printDownloadDebugInfo(ip, nil, response.StatusCode, downloadURL, lastRedirectURL, response)
		}
		return 0.0, ""
	}

	// 通过头部参数获取地区码
	colo := getHeaderColo(response.Header)
	contentLength := response.ContentLength // 文件大小

	body, err := io.ReadAll(response.Body) // 10 s 内必须读完
	if err != nil {
		return 0.0, ""
	}
	if len(body) != int(contentLength) {
		return 0.0, ""
	}
	costTime := time.Since(timeStart)
	return float64(contentLength) / costTime.Seconds(), colo
}

func (s *SpeedResult) DownloadTest(
	downloadTestTimes int,
	downloadTimeOut time.Duration,
	downloadURL string,
	downloadTCPPort int,
	bar *utils.Bar) {
	defer bar.Grow(1, "")
	s.DownloadSpeed = 0
	var totalSpeed float64 = 0
	for i := 0; i < downloadTestTimes; i++ {
		speed, colo := downloadURLByIP(downloadTimeOut, downloadURL, s.IP, downloadTCPPort)
		totalSpeed += speed
		if s.Colo == "" { // 只有当 Colo 是空的时候，才写入，否则代表之前是 httping 测速并获取过了
			s.Colo = colo
		}
	}
	s.DownloadSpeed = totalSpeed / float64(downloadTestTimes)
}

func (s *SpeedResultSlice) DownloadTest(
	downloadTestIPNum int,
	downloadIPTestTimes int,
	downloadTimeout time.Duration,
	downloadURL string,
	downloadTCPPort int) {
	bar := utils.NewBar(downloadTestIPNum, "", "")
	if downloadTestIPNum > len(*s) {
		downloadTestIPNum = len(*s)
	}
	for i := 0; i < downloadTestIPNum; i++ {
		(*s)[i].DownloadTest(downloadIPTestTimes, downloadTimeout, downloadURL, downloadTCPPort, bar)
	}
	bar.Done()
}
