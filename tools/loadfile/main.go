package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

func main() {
	fi, err := os.Open("./res.log")
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
		if err := GetWeb(string(a)); err != nil {
			break
		}
		count = count + 1
		time.Sleep(time.Millisecond * 200)
	}
	fmt.Printf("\nall count:%d\n", count)
}

func GetWeb(url string) error {
	dt := strings.Split(url, " ")
	if len(dt) > 4 {
		dd := dt[len(dt)-1]
		fmt.Printf("dd:%v\n", dd)
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:2003/%v", dd))
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
	}
	return nil
}
