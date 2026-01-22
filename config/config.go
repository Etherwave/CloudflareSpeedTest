package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	Version               = "1.0.0"
	WindowsHostsFilePath  = "C:\\Windows\\System32\\drivers\\etc\\hosts"
	LinuxHostsFilePath    = "/etc/hosts"
	ConfigFilePath        = "config.json"
	Debug                 = false
	MaxAllowDelay         = time.Duration(300 * time.Millisecond)
	MaxDelay              = time.Duration(9999 * time.Millisecond)
	MaxLossRate           = 1.0 // 100%
	TestAllowIPV4NumRatio = 0.1
)

var (
	appExecPath    string
	appBaseDir     string
	configInitOnce sync.Once
	initErr        error

	Config ConfigJson
	Rand   *rand.Rand
)

type StrSet map[string]struct{}

func (s StrSet) Add(v string)           { s[v] = struct{}{} }
func (s StrSet) Contains(v string) bool { _, ok := s[v]; return ok }

// ---------- JSON 支持 ----------
func (s *StrSet) MarshalJSON() ([]byte, error) {
	keys := make([]string, 0, len(*s))
	for k := range *s {
		keys = append(keys, k)
	}
	return json.Marshal(keys)
}

func (s *StrSet) UnmarshalJSON(data []byte) error {
	if *s == nil {
		*s = make(StrSet)
	}
	var keys []string
	if err := json.Unmarshal(data, &keys); err != nil {
		return err
	}
	for _, k := range keys {
		(*s)[k] = struct{}{}
	}
	return nil
}

type ConfigJson struct {
	// base config
	OutputFile         string   `json:"OutputFile"`
	TestMode           string   `json:"TestMode"`
	EnableDownLoadTest bool     `json:"EnableDownLoadTest"`
	FastTest           bool     `json:"FastTest"`
	WebHosts           []string `json:"WebHosts"`
	TestIPNum          int      `json:"TestIPNum"` // set -1 if test all IPs
	SaveIPNum          int      `json:"SaveIPNum"`
	// ip config
	CIDRIPV4File    string `json:"CIDRIPV4File"` // https://www.cloudflare.com/ips-v4
	CIDRIPV6File    string `json:"CIDRIPV6File"` // https://www.cloudflare.com/ips-v6
	AllowIPV4RBFile string `json:"AllowIPV4RBFile"`
	DenyIPV4RBFile  string `json:"DenyIPV4RBFile"`
	// tcp config
	TcpRoutines       int           `json:"TcpRoutines"`
	TcpPort           int           `json:"TcpPort"`
	TcpConnectTimes   int           `json:"TcpConnectTimes"`
	TcpConnectTimeout time.Duration `json:"TcpConnectTimeout"`
	// http config
	HttpColo           string        `json:"HttpColo"`
	HttpColoSet        StrSet        `json:"HttpColoSet"`
	HttpConnectTimes   int           `json:"HttpConnectTimes"`
	HttpConnectTimeout time.Duration `json:"HttpConnectTimeout"`
	HttpRoutines       int           `json:"HttpRoutines"`
	HttpStatusCode     int           `json:"HttpStatusCode"`
	HttpURL            string        `json:"HttpURL"`
	HttpTCPPort        int           `json:"HttpTCPPort"`
	// download config
	DownloadTestIPNum   int           `json:"DownloadTestIPNum"`
	DownloadIPTestTimes int           `json:"DownloadIPTestTimes"`
	DownloadTimeout     time.Duration `json:"DownloadTimeout"`
	DownloadURL         string        `json:"DownloadURL"`
	DownloadTCPPort     int           `json:"DownloadTCPPort"`
}

func NewConfigJson() *ConfigJson {
	return &ConfigJson{
		OutputFile:          "results.csv",
		TestMode:            "tcp", // tcp or http
		EnableDownLoadTest:  true,
		FastTest:            false,
		WebHosts:            []string{},
		TestIPNum:           100,
		SaveIPNum:           100,
		CIDRIPV4File:        "ip.txt",
		CIDRIPV6File:        "ipv6.txt",
		AllowIPV4RBFile:     "allow_ipv4.rb",
		DenyIPV4RBFile:      "deny_ipv4.rb",
		TcpRoutines:         30,
		TcpPort:             443,
		TcpConnectTimes:     3,
		TcpConnectTimeout:   2 * time.Second,
		HttpColo:            "",
		HttpColoSet:         nil,
		HttpConnectTimes:    3,
		HttpConnectTimeout:  5 * time.Second,
		HttpRoutines:        10,
		HttpStatusCode:      200,
		DownloadTestIPNum:   10,
		DownloadIPTestTimes: 1,
		DownloadTimeout:     3 * time.Second,
		DownloadURL:         "",
		DownloadTCPPort:     443,
	}
}

func (c *ConfigJson) Load(configFilePath string) error {
	if configFilePath == "" {
		return errors.New("config file path is empty")
	}
	// read config file
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return err
	}
	// parse config file
	err = json.Unmarshal(data, c)
	if err != nil {
		return err
	}
	return nil
}

func (c *ConfigJson) Save(configFilePath string) error {
	if configFilePath == "" {
		configFilePath = filepath.Join(appBaseDir, ConfigFilePath)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(configFilePath, data, 0644)
}

func loadConfigJson(configFilePath string) error {
	Config = *NewConfigJson()
	defer Config.Save(configFilePath)
	var err error
	if configFilePath == "" {
		configFilePath = filepath.Join(appBaseDir, ConfigFilePath)
	}
	_, errExist := os.Stat(configFilePath)
	if os.IsNotExist(errExist) {
		return nil
	}
	err = Config.Load(configFilePath)
	if err != nil {
		return err
	}
	return nil
}

// InitPaths 初始化路径（在程序启动时调用一次）
func InitPaths() error {
	var err error
	var exePath string
	exePath, err = os.Executable()
	if err != nil {
		return err
	}
	appExecPath, err = filepath.Abs(exePath)
	if err != nil {
		return err
	}
	appBaseDir = filepath.Dir(appExecPath)
	return err
}

// GetExecPath 获取可执行文件完整路径
func GetExecPath() string {
	return appExecPath
}

// GetBaseDir 获取可执行文件所在目录
func GetBaseDir() string {
	return appBaseDir
}
func initRand() {
	// 用 PCG，给两个 64 位种子（随便写，只要不同运行不同即可）
	src := rand.NewPCG(uint64(rand.Uint64()), uint64(rand.Uint64()))
	Rand = rand.New(src)
}

func Init() error {
	configInitOnce.Do(func() {
		initErr = doInit()
	})
	return initErr
}

func doInit() error {
	var err error
	initRand()
	err = InitPaths()
	if err != nil {
		return err
	}
	var configFilePath = flag.String("config", "config.json", "path to config file")
	var testIPNum = flag.Int("n", -1, "number of IPs to test")
	flag.Parse()

	fmt.Println("config:", *configFilePath)
	err = loadConfigJson(*configFilePath)
	if *testIPNum != -1 {
		Config.TestIPNum = *testIPNum
	}
	return err
}
