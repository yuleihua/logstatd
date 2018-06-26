package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	//	"net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	//"github.com/json-iterator/go"
	"airman.com/logstatd/pkg/conf"
	"airman.com/logstatd/pkg/metric"
	st "airman.com/logstatd/pkg/stat"
	log "github.com/sirupsen/logrus"
)

const (
	defaultDeadline = 10 * time.Second
	defaultMetric   = 60 * time.Second
	DataStr         = "data"
	ModeStr         = "mode"
	TopicStr        = "topic"
)

var confname string

func init() {
	flag.StringVar(&confname, "c", "conf/logstatd.ini", "configure file")
}

func getMachineId(addr string) int64 {
	var tmpstr string
	ipList := strings.Split(addr, ":")
	if len(ipList) <= 1 || ipList[0] == "" || ipList[0] == "*" {
		//default
		tmpstr = os.Getenv("ENV_MID")
	} else {
		tmpstr = strings.Split(ipList[0], ".")[3]
	}
	mid, err := strconv.ParseInt(tmpstr, 10, 64)
	if err != nil {
		//default
		log.Fatalf("invalid ip string:%v", tmpstr)
	}
	return mid
}

type LogMessage struct {
	Channel   string
	Appid     string
	AppVerson string `json:"app_version"`
	ClientIp  string `json:"client_ip"`
	Source    string
	Type      string
	SysInfo   string `json:"sys_channel"`
	UserInfo  string `json:"user_info"`
	PageInfo  string `json:"page_info"`
	EventInfo string `json:"event_info"`
	ExtInfo   string `json:"ext_info"`
}

func logMsgData(w http.ResponseWriter, r *http.Request) {
	var ms []LogMessage
	if err := ParseJson(r, &ms); err != nil {
		panic(err)
	}

	_, month, day := time.Now().Date()
	count := 0
	for _, m := range ms {
		topic := fmt.Sprintf("%s_%d%d", m.Type, month, day)
		dataval, err := json.Marshal(m)
		if err != nil {
			log.Errorf("json marshal error:%v,data(%v)", err, m)
			continue
		}
		GetServer().TransportMsg(topic, string(dataval), 0, true)
		count = count + 1
	}
	WriteResponse(w, http.StatusOK, count)
}

func WriteResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Errorf("json encode body error, %v:%v", data, err)
		return
	}
}

func ParseJson(req *http.Request, obj interface{}) error {
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(obj); err != nil {
		log.Errorf("json decode body error, %v:%v", req.Body, err)
		return err
	}
	return nil
}

type LogKafka struct {
	Topic string
	Data  string
	Type  string
}

func kafkaHandler(w http.ResponseWriter, r *http.Request) {
	st.GetStat().Inc(st.ST_REQ)

	var ms []LogKafka
	if err := ParseJson(r, &ms); err != nil {
		panic(err)
	}

	count := 0
	for _, m := range ms {
		GetServer().TransportMsg(m.Topic, m.Data, 0, true)
		count = count + 1
	}
	WriteResponse(w, http.StatusOK, count)
}

func statHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, st.GetStat().NowStat())
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, st.GetStat().GetStatus())
}

func uuidHandler(w http.ResponseWriter, r *http.Request) {
	uuid, err := GetServer().GetUid()
	if err != nil {
		log.Errorf("uuid error, %v", err)
		return
	}
	WriteResponse(w, http.StatusOK, uuid)
}

func main() {
	flag.Parse()

	c := make(chan os.Signal)
	signal.Ignore()
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	// read configure file
	conf.Setup(confname)

	// setting logger
	log.SetFormatter(&log.TextFormatter{
		DisableColors:   true,
		TimestampFormat: "2006/01/02-15:04:05.000",
	})

	logfile := conf.GetLogger().LogFile
	if logfile != "" {
		if logfile != "console" {
			f, err := os.OpenFile(conf.GetLogger().LogFile, os.O_CREATE|os.O_WRONLY, 0666)
			if err == nil {
				log.SetOutput(f)
			} else {
				log.Warnf("Failed to log to file(%s), using default stderr", conf.GetLogger().LogFile)
			}
		}
		log.SetLevel(log.Level(conf.GetLogger().LogLevel))
	}

	// handle
	if err := NewServer(conf.GetService(), conf.GetProxy()); err != nil {
		log.Fatalf("Start server error:%v", err)
	}

	router := http.NewServeMux()
	router.HandleFunc("/addkafka", kafkaHandler)
	router.HandleFunc("/logkafka", logMsgData)
	router.Handle("/metric", metric.WebMetricHandler(defaultMetric))
	router.HandleFunc("/stats", statHandler)
	router.HandleFunc("/status", statusHandler)
	router.HandleFunc("/idgen", uuidHandler)

	srv := &http.Server{
		Addr:         conf.GetService().SrvAddr,
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	<-c

	// shutdown http server
	log.Error("shutting down http server with graceful")
	ctx, _ := context.WithTimeout(context.Background(), defaultDeadline)
	srv.Shutdown(ctx)

	log.Error("shutting down worker server")
	if err := GetServer().Shutdown(); err != nil {
		log.Errorf("shutdown error:%v", err)
	}
	log.Error("shutting down end")
}
