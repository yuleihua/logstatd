package kafka

import (
	"strings"
	"time"

	"github.com/Shopify/sarama"
	log "github.com/sirupsen/logrus"
)

type KafkaClient struct {
	sarama.SyncProducer
	Address   string
	Timeout   time.Duration
	isSuccess bool
}

func NewKafkaClient(address string, timeout time.Duration, isSuccess bool) (*KafkaClient, error) {
	config := sarama.NewConfig()
	// config.Producer.RequiredAcks = sarama.WaitForAll
	// NewRandomPartitioner ;NewRoundRobinPartitioner ;NewRandomPartitioner ;NewHashPartitioner
	config.Net.DialTimeout = timeout
	config.Producer.Partitioner = sarama.NewRoundRobinPartitioner
	config.Producer.Return.Successes = isSuccess
	config.Producer.Timeout = timeout

	producer, err := sarama.NewSyncProducer(strings.Split(address, ","), config)
	if err != nil {
		log.Errorf("NewProducer error:%v", err)
		return nil, err
	}

	return &KafkaClient{
		SyncProducer: producer,
		Address:      address,
		Timeout:      timeout,
		isSuccess:    isSuccess,
	}, nil
}

func (p *KafkaClient) SendMsgByByte(topic string, msg string) error {
	m := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(msg),
	}

	if _, _, err := p.SendMessage(m); err != nil {
		log.Errorf("send kafka error, %v", err)
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
		log.Errorf("send kafka error, %v", err)
		return err
	}
	return nil
}

func (p *KafkaClient) SendMsg(msg *sarama.ProducerMessage) error {
	if msg != nil {
		if _, _, err := p.SendMessage(msg); err != nil {
			log.Errorf("send kafka error, %v", err)
			return err
		}
	}
	return nil
}

func (p *KafkaClient) SendMsgs(msgs []*sarama.ProducerMessage) error {
	if len(msgs) > 0 {
		if err := p.SendMessages(msgs); err != nil {
			log.Errorf("send kafka error, %v", err)
			return err
		}
	}
	return nil
}

func (p *KafkaClient) Close() error {
	if err := p.SyncProducer.Close(); err != nil {
		log.Errorf("close syncproducer error, %v", err)
		return err
	}
	return nil
}
