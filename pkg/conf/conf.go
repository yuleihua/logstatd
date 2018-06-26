package conf

import (
	"github.com/BurntSushi/toml"
	log "github.com/sirupsen/logrus"
)

var Config Conf

type Conf struct {
	Logger    *Logger    `toml:"log"`
	Service   *Service   `toml:"service"`
	Proxy     *Proxy     `toml:"proxy"`
	RegServer *RegServer `toml:"register"`
}

type Logger struct {
	LogLevel  int32  `toml:"log_level"`
	LogRotate bool   `toml:"log_rotate"`
	LogFile   string `toml:"log_file"`
}

type Service struct {
	SrvAddr   string `toml:"addr"`
	MonAddr   string `toml:"mon_addr"`
	IsProxy   bool   `toml:"isproxy"`
	PoolMin   int    `toml:"pool_min"`
	PoolMax   int    `toml:"pool_max"`
	WorkerNum int    `toml:"worke_num"`
	Interval  int    `toml:"interval"`
	JobqNum   int    `toml:"jobq_num"`
	FileName  string `toml:"leveldb_name"`
}

type Proxy struct {
	Proxy   []string `toml:"kafkas"`
	Timeout int      `toml:"timeout"`
}

type RegServer struct {
	addrs   []string `toml:"addrs"`
	Timeout int      `toml:"timeout"`
}

func GetLogger() *Logger {
	return Config.Logger
}

func GetProxy() *Proxy {
	return Config.Proxy
}

func GetService() *Service {
	return Config.Service
}

func GetRegServer() *RegServer {
	return Config.RegServer
}

func Setup(fileName string) Conf {
	if _, err := toml.DecodeFile(fileName, &Config); err != nil {
		log.Fatalf("%v", err)
	}
	return Config
}
