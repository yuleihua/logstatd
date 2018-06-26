package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func doTopicFile(topic, folder string) {
	files, _ := ioutil.ReadDir(folder) //specify the current dir
	for _, file := range files {
		if !file.IsDir() {
			fmt.Println(folder + "/" + file.Name())
			go handleFile(topic, fmt.Sprintf("%v/%v", folder, file.Name()))
		}
	}
}

func handleFile(topic, myfile string) {
	fi, err := os.Open("myfile")
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}
	defer fi.Close()

	var count int
	br := bufio.NewReader(fi)
	for {
		a, _, c := br.ReadLine()
		if c == io.EOF {
			break
		}
		//fmt.Println(string(a))
		if err := getWeb(topic, string(a)); err != nil {
			break
		}
		count = count + 1
		time.Sleep(time.Millisecond * 200)
	}
	fmt.Printf("\nall file:%v count:%d\n", myfile, count)
}

func getWeb(topic, data string) error {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:2003/express?data=%v&upstream=kafka&async=1&topic=%v", data, topic))
	if err != nil {
		// handle error
		fmt.Printf("Error: %s\n", err)
		return err
	}
	defer resp.Body.Close()

	if _, err := ioutil.ReadAll(resp.Body); err != nil {
		fmt.Printf("Error: %s\n", err)
		return err
	}
	return nil
}

var (
	pathname string
	topic    string
)

func init() {
	flag.StringVar(&pathname, "p", "./data", "topic data file")
	flag.StringVar(&topic, "t", "read", "topic")
}

func main() {
	flag.Parse()

	c := make(chan os.Signal)
	signal.Ignore()
	signal.Notify(c, syscall.SIGTERM, os.Interrupt)

	fmt.Printf("topic info:(%v:%v) \n", topic, pathname)
	doTopicFile(topic, pathname)

	<-c
	fmt.Println("shutting down")
}
