package pubsub

import (
	"sync"

	"github.com/PovilasDuo/Infocenter-service/message"
)

type Topic struct {
	Subscribers  map[chan message.Message]struct{}
	Messages     []message.Message
	MessageQueue []message.Message
	Mu           sync.RWMutex
}

var (
	topics   = make(map[string]*Topic)
	topicsMu sync.RWMutex
)

func GetOrCreateTopic(name string) *Topic {
	topicsMu.Lock()
	defer topicsMu.Unlock()

	if topic, exists := topics[name]; exists {
		return topic
	}

	newTopic := &Topic{
		Subscribers:  make(map[chan message.Message]struct{}),
		Messages:     []message.Message{},
		MessageQueue: []message.Message{},
	}
	topics[name] = newTopic
	return newTopic
}
