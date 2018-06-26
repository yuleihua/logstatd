package kafka

import (
	"time"

	"github.com/Shopify/sarama"
	log "github.com/sirupsen/logrus"
)

type KafkaClient struct {
	sarama.SyncProducer
	Addrs    []string
	Timeout  time.Duration
	IsSucess bool
}

func NewKafkaClient(addrs []string, timeout time.Duration, isSucess bool) (*KafkaClient, error) {
	config := sarama.NewConfig()
	// config.Producer.RequiredAcks = sarama.WaitForAll
	// NewRandomPartitioner ;NewRoundRobinPartitioner ;NewRandomPartitioner ;NewHashPartitioner
	config.Net.DialTimeout = timeout
	config.Producer.Partitioner = sarama.NewRoundRobinPartitioner
	config.Producer.Return.Successes = isSucess
	config.Producer.Timeout = timeout
	producer, err := sarama.NewSyncProducer(addrs, config)
	if err != nil {
		log.Errorf("NewProducer error:%v", err)
		return nil, err
	}

	return &KafkaClient{
		SyncProducer: producer,
		Addrs:        addrs,
		Timeout:      timeout,
		IsSucess:     isSucess,
	}, nil
}

func (p *KafkaClient) SendMsgByByte(topic string, msg string) error {
	m := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(msg),
	}

	if _, _, err := p.SendMessage(m); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (p *KafkaClient) SendMsgByString(topic string, msg string) error {
	m := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.StringEncoder(msg),
	}

	if _, _, err := p.SendMessage(m); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

func (p *KafkaClient) SendMsg(msg *sarama.ProducerMessage) error {
	if msg != nil {
		if _, _, err := p.SendMessage(msg); err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}

func (p *KafkaClient) SendMsgs(msgs []*sarama.ProducerMessage) error {
	if len(msgs) > 0 {
		if err := p.SendMessages(msgs); err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}

func (p *KafkaClient) Close() error {
	if e := p.SyncProducer.Close(); e != nil {
		log.Error("close syncproducer error:", e)
		return e
	}
	return nil
}
