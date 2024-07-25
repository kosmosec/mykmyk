package sns

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kosmosec/mykmyk/internal/model"
)

func TestCreateTopic(t *testing.T) {
	topic1 := "topic1"
	topic2 := "topic2"

	sns := New()

	sns.CreateTopic(topic1)
	sns.CreateTopic(topic2)

	_, ok := sns.topics[topic1]
	if !ok {
		t.Fatalf("topic not found")
	}

	_, ok = sns.topics[topic2]
	if !ok {
		t.Fatalf("topic not found")
	}

}

func TestAddConsumer(t *testing.T) {
	topic1 := "topic1"
	consumerName := "consumerTest"
	consumer1 := make(chan model.Message, 1)
	consumer2 := make(chan model.Message, 1)

	sns := New()
	sns.CreateTopic(topic1)

	sns.AddConsumer(topic1, consumerName, consumer1)
	sns.AddConsumer(topic1, consumerName, consumer2)

	addedConsumers := [](chan model.Message){consumer1, consumer2}

	topic := sns.topics[topic1]

	for i, c := range topic.consumers {
		if addedConsumers[i] != c.consumer {
			t.Fatalf("invalid consumer")
		}

	}
}

func TestSendMessage(t *testing.T) {
	topic1 := "topic1"
	consumerName := "consumerTest"
	consumer1 := make(chan model.Message, 1)
	consumer2 := make(chan model.Message, 1)

	sns := New()
	sns.CreateTopic(topic1)

	sns.AddConsumer(topic1, consumerName, consumer1)
	sns.AddConsumer(topic1, consumerName, consumer2)

	msg := model.Message{Targets: []string{"127.0.0.1"}, Ports: []string{"8080"}}
	sns.SendMessage(topic1, msg)

	rcvMessage1 := <-consumer1
	rcvMessage2 := <-consumer2

	if cmp.Diff(rcvMessage1, msg) != "" || cmp.Diff(rcvMessage2, msg) != "" {
		t.Fatalf("message are different")
	}
}
