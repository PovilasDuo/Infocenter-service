package pubsub

import (
	"sync"

	"github.com/PovilasDuo/Infocenter-service/message"
)

const MaxMessages = 1000

type PubSub struct {
	topics    map[string]*Topic
	topicsMu  sync.RWMutex
	messageID int
	messageMu sync.Mutex
}

func NewPubSub() *PubSub {
	return &PubSub{
		topics: make(map[string]*Topic),
	}
}

func (ps *PubSub) NextMessageID() int {
	ps.messageMu.Lock()
	defer ps.messageMu.Unlock()
	ps.messageID++
	return ps.messageID
}

func (ps *PubSub) GetOrCreateTopic(name string) *Topic {
	ps.topicsMu.Lock()
	defer ps.topicsMu.Unlock()

	if topic, exists := ps.topics[name]; exists {
		return topic
	}

	newTopic := NewTopic()
	ps.topics[name] = newTopic
	return newTopic
}

func (ps *PubSub) AddSubscriber(topic *Topic, msgCh chan message.Message) {
	topic.Mu.Lock()
	defer topic.Mu.Unlock()
	topic.Subscribers[msgCh] = struct{}{}
}

func (ps *PubSub) RemoveSubscriber(topic *Topic, msgCh chan message.Message) {
	topic.Mu.Lock()
	defer topic.Mu.Unlock()
	delete(topic.Subscribers, msgCh)
	close(msgCh)
}
