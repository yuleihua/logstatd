package kafka

import (
	"fmt"
	"testing"
	"time"
	//	"github.com/stretchr/testify/assert"
)

func TestClientPool(t *testing.T) {
	fn := func() (*KafkaClient, error) {
		return NewKafkaClient([]string{"43.247.89.156:9092"}, 15*time.Second, true)
	}

	ts, err := NewClientPool(2, 4, fn)
	if err != nil {
		t.Fatalf("NewClientPool error:%v", err)
	}

	for i := 0; i < 2; i++ {
		cli, err := ts.Get()
		if err != nil {
			t.Fatalf("Get error")
		}

		docli := func(cli *WrapClient) {
			if err := cli.SendMsgByByte("venus", "value"); err != nil {
				t.Fatalf("SendMsgByByte error")
			}

			if err := cli.SendMsgByString("venus", "8712t7832gds7dfg7dfg7t7#@@##0000%#@#"); err != nil {
				t.Fatalf("SendMsgByByte error")
			}
			time.Sleep(time.Duration(3+i) * time.Second)
			cli.Close()
		}
		go docli(cli)
	}

	fmt.Printf("11 len:%v\n", ts.Len())
	time.Sleep(4 * time.Second)
	fmt.Printf("22 len:%v\n", ts.Len())

	for i := 0; i < 4; i++ {
		cli, err := ts.Get()
		if err != nil {
			t.Fatalf("Get error")
		}

		docli := func(cli *WrapClient) {
			if err := cli.SendMsgByByte("venus", "value"); err != nil {
				t.Fatalf("SendMsgByByte error")
			}

			if err := cli.SendMsgByString("venus", "8712t7832gds7dfg7dfg7t7#@@##0000%#@#"); err != nil {
				t.Fatalf("SendMsgByByte error")
			}
			time.Sleep(3 * time.Second)
			cli.Close()
		}
		go docli(cli)
	}
	fmt.Printf("33 len:%v\n", ts.Len())
	time.Sleep(5 * time.Second)
	fmt.Printf("44 len:%v\n", ts.Len())

	ts.Close()
}
