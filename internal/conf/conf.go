package conf

import (
	"encoding/json"
	"io/ioutil"

	log "github.com/sirupsen/logrus"
)

var Config Conf

type Conf struct {
	Service *Service `json:"service"`
	Kafka   *Kafka   `json:"kafka"`
}

type Service struct {
	Address   string `json:"address"`
	LogFile   string `json:"log_file,omitempty"`
	LogLevel  int    `json:"log_level,omitempty"`
	PoolMin   int    `json:"pool_min"`
	PoolMax   int    `json:"pool_max"`
	WorkerNum int    `json:"worker_num"`
	Interval  int    `json:"interval"`
	JobNum    int    `json:"queue_num"`
	DBName    string `json:"leveldb_name"`
}

type Kafka struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Timeout int      `json:"timeout,omitempty"`
}

func GetKafka() *Kafka {
	return Config.Kafka
}

func GetService() *Service {
	return Config.Service
}

func Setup(fileName string) *Conf {
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Fatalf("open config error, %s, %v", fileName, err)
	}

	if err := json.Unmarshal(content, &Config); err != nil {
		log.Fatalf("config json unmarshal %v", err)
	}

	return &Config
}
