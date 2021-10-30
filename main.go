package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fasthttp/router"
	jsoniter "github.com/json-iterator/go"
	log "github.com/sirupsen/logrus"
	"github.com/valyala/fasthttp"
	"github.com/yuleihua/logstatd/internal/conf"
	"github.com/yuleihua/logstatd/internal/middleware"
	"github.com/yuleihua/logstatd/internal/server"
	"github.com/yuleihua/logstatd/internal/stat"
)

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary

	configFile string
	serverMain *server.Server
)

func init() {
	flag.StringVar(&configFile, "f", "conf/logstatd.json", "configure file")
}

func main() {
	flag.Parse()

	// read configure file
	conf.Setup(configFile)

	// setting logger
	log.SetFormatter(&log.TextFormatter{
		DisableColors:   true,
		TimestampFormat: "2006/01/02-15:04:05.000",
	})

	logfile := conf.GetService().LogFile
	if logfile == "" {
		logfile = "console"
	}

	if logfile != "" && logfile != "console" {
		f, err := os.OpenFile(logfile, os.O_CREATE|os.O_WRONLY, 0666)
		if err == nil {
			log.SetOutput(f)
		} else {
			log.Warnf("Failed to log to file(%s), using default stderr", logfile)
		}
	}

	level := conf.GetService().LogLevel
	if level == 0 {
		level = 5
	}
	log.SetLevel(log.Level(level))

	c := make(chan os.Signal, 1)
	signal.Ignore()
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

	// handle
	server, err := server.NewServer(conf.GetService(), conf.GetKafka())
	if err != nil {
		log.Fatalf("Start server error:%v", err)
	}
	serverMain = server

	router := router.New()
	mds := middleware.DefaultMiddlewares

	if conf.GetService().IsCors && len(conf.GetService().Origins) > 0 {
		mds = mds.Append(middleware.NewCorsHandler(*middleware.NewOption(conf.GetService().Origins)))
	}

	if conf.GetService().IsAuth && len(conf.GetService().Auths) > 0 {
		mds = mds.Append(middleware.NewAuthorization(conf.GetService().Auths[0].AppId, conf.GetService().Auths[0].AppSecret, conf.GetService().Auths[0].AppTimeout))
	}

	if conf.GetService().IsPrometheus {
		mds = mds.Append(middleware.NewPrometheus("logstatd", "/metrics"))
	}

	router.POST("/kafka", mds.Apply(Kafka))
	router.POST("/channel", mds.Apply(LogChannel))
	router.GET("/status", mds.Apply(Status))
	router.GET("/uuid", mds.Apply(UUID))

	errChan := make(chan error, 16)
	go func() {
		if err := fasthttp.ListenAndServe(conf.GetService().Address, router.Handler); err != nil {
			log.Println(err)
			errChan <- err
		}
	}()

	select {
	case <-c:
		log.Warnf("receive quit signal")
		break
	case e := <-server.ErrChan:
		log.Errorf("stack error, %v", e)
		break
	case e := <-errChan:
		log.Errorf("stack error, %v", e)
		break
	}

	log.Error("shutting down worker server")
	if err := serverMain.Shutdown(); err != nil {
		log.Errorf("shutdown error:%v", err)
	}
	log.Error("shutting down end")

	//ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	//defer cancel()
	//
	//if config.CommonConfig().MetricAddress != "" && metricServer != nil {
	//	metricServer.Shutdown(ctx)
	//}

	// shutdown http server
	log.Error("shutting down worker begin")
	if err := serverMain.Shutdown(); err != nil {
		log.Errorf("shutdown worker error:%v", err)
	}
	log.Error("shutting down end")
}

type LogMessage struct {
	Source  string   `json:"source,omitempty"`
	Channel string   `json:"channel"`
	Items   []string `json:"items"`
}

func LogChannel(ctx *fasthttp.RequestCtx) {
	if !bytes.Equal(ctx.Method(), []byte(fasthttp.MethodPost)) {
		ctx.SetStatusCode(fasthttp.StatusForbidden)
		return
	}

	if len(ctx.PostBody()) <= 2 {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	var ms LogMessage
	if err := json.Unmarshal(ctx.PostBody(), &ms); err != nil {
		log.Errorf("json error, %s, %v", string(ctx.PostBody()), err)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	count := 0
	for _, msg := range ms.Items {
		serverMain.TransportMsg("", msg, true)
		count = count + 1
	}

	data := fmt.Sprintf("{\"%s\":%d}", "result", count)
	if _, err := ctx.Write([]byte(data)); err != nil {
		log.Errorf("write error, %v", err)
		return
	}

	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
}

type KafkaMsg struct {
	Topic string   `json:"topic,omitempty"`
	Items []string `json:"items"`
	Type  string   `json:"type,omitempty"`
}

func Kafka(ctx *fasthttp.RequestCtx) {
	if !bytes.Equal(ctx.Method(), []byte(fasthttp.MethodPost)) {
		ctx.SetStatusCode(fasthttp.StatusForbidden)
		return
	}

	if len(ctx.PostBody()) <= 2 {
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	var ms KafkaMsg
	if err := json.Unmarshal(ctx.PostBody(), &ms); err != nil {
		log.Errorf("json error, %s, %v", string(ctx.PostBody()), err)
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		return
	}

	count := 0
	for _, msg := range ms.Items {
		serverMain.TransportMsg(ms.Topic, msg, true)
		count = count + 1
	}

	data := fmt.Sprintf("{\"%s\":%d}", "result", count)

	if _, err := ctx.Write([]byte(data)); err != nil {
		log.Errorf("write error, %v", err)
		return
	}

	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
}

func Status(ctx *fasthttp.RequestCtx) {
	s := stat.GetStat()

	content, err := json.Marshal(s)
	if err != nil {
		log.Errorf("json marshal error, %v, %v", s, err)
		return
	}

	if _, err = ctx.Write(content); err != nil {
		log.Errorf("write error, %v", err)
		return
	}

	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	return
}

type IdCode struct {
	Id int64 `json:"uuid"`
}

func UUID(ctx *fasthttp.RequestCtx) {
	if serverMain == nil {
		return
	}

	uuid, err := serverMain.GetUUid()
	if err != nil {
		log.Errorf("uuid error, %v", err)
		return
	}

	data := &IdCode{
		Id: uuid,
	}

	content, err := json.Marshal(data)
	if err != nil {
		log.Errorf("json marshal error, %v, %v", data, err)
		return
	}

	if _, err = ctx.Write(content); err != nil {
		log.Errorf("write error, %v", err)
		return
	}

	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.SetStatusCode(fasthttp.StatusOK)
	return
}
