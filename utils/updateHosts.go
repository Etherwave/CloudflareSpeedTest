package utils

import (
	"CloudflareSpeedTest/config"
	"bufio"
	"os"
	"runtime"
	"strings"
)

type StringSet map[string]struct{}

func (s StringSet) Add(v string)           { s[v] = struct{}{} }
func (s StringSet) Contains(v string) bool { _, ok := s[v]; return ok }

func getHostsFilePath() string {
	var hostsFilePath string
	if runtime.GOOS == "windows" {
		hostsFilePath = config.WindowsHostsFilePath
	} else {
		hostsFilePath = config.LinuxHostsFilePath
	}
	return hostsFilePath
}
func UpdateHosts(ip string, hosts []string) error {
	if len(hosts) == 0 {
		return nil
	}
	lowerHostsSet := make(StringSet)
	for _, host := range hosts {
		lowerHostsSet.Add(strings.ToLower(host))
	}
	hostsFilePath := getHostsFilePath()
	lines, err := readHostsFile(hostsFilePath)
	if err != nil {
		return err
	}
	for i, line := range lines {
		// 跳过空行和注释
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 只取前两项：ip  host
		fields := strings.Fields(line)
		if len(fields) < 2 || !IsIPv4(fields[0]) {
			continue
		}
		_, host := fields[0], fields[1]
		lowerHost := strings.ToLower(host)
		if lowerHostsSet.Contains(lowerHost) {
			lines[i] = ip + " " + lowerHost // 替换
		}
	}
	err = writeHostsFile(hostsFilePath, lines)
	return err
}

func readHostsFile(hostsFilePath string) ([]string, error) {
	file, err := os.Open(hostsFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, strings.TrimSpace(line))
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

func writeHostsFile(hostsFilePath string, lines []string) error {
	file, err := os.Create(hostsFilePath)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return err
		}
	}
	return writer.Flush()
}
