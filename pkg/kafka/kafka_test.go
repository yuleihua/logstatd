package kafka

import (
	"testing"
	"time"
)

func TestKafkaBasic(t *testing.T) {
	ts, err := NewKafkaClient([]string{"43.247.89.156:9092"}, 10*time.Second, true)
	defer ts.Close()
	if err != nil {
		t.Fatalf("NewKafkaClient error:%v", err)
	}

	if err := ts.SendMsgByByte("venus", "value"); err != nil {
		t.Fatalf("SendMsgByByte error")
	}

	if err := ts.SendMsgByString("venus", "8712t7832gds7dfg7dfg7t7#@@##0000%#@#"); err != nil {
		t.Fatalf("SendMsgByByte error")
	}
}
