package sns

import (
	"github.com/kosmosec/mykmyk/internal/model"
)

type SNS struct {
	topics map[string]topic
}

type topic struct {
	name      string
	consumers []consumer
}

type consumer struct {
	name     string
	consumer chan model.Message
}

func New() SNS {
	return SNS{
		topics: make(map[string]topic),
	}
}

func (s *SNS) CreateTopic(name string) {
	t := topic{
		name: name,
	}
	s.topics[name] = t
}

func (s *SNS) AddConsumer(topic string, consumerName string, consumerCh chan model.Message) {
	t := s.topics[topic]
	c := consumer{
		name:     consumerName,
		consumer: consumerCh,
	}
	t.consumers = append(t.consumers, c)
	s.topics[topic] = t
}

func (s *SNS) SendMessage(topic string, msg model.Message) {
	t := s.topics[topic]
	for _, c := range t.consumers {
		c.consumer <- msg
	}
}

func (s *SNS) GetTopic(topic string, consumerName string) chan model.Message {
	t := s.topics[topic]
	for _, c := range t.consumers {
		if c.name == consumerName {
			return c.consumer
		}
	}
	return nil
}

func (s *SNS) CloseTopic(topic string) {
	t := s.topics[topic]
	for _, c := range t.consumers {
		if c.consumer != nil {
			close(c.consumer)
		}
	}

}
