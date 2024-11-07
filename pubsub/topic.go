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

func NewTopic() *Topic {
	return &Topic{
		Subscribers:  make(map[chan message.Message]struct{}),
		Messages:     []message.Message{},
		MessageQueue: []message.Message{},
	}
}
