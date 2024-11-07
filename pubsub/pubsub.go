package pubsub

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"

	"github.com/PovilasDuo/Infocenter-service/message"
)

const MaxMessages = 1000

type PubSub struct {
	topics    map[string]*Topic
	topicsMu  sync.RWMutex
	messageID int32
}

func NewPubSub() *PubSub {
	return &PubSub{
		topics: make(map[string]*Topic),
	}
}

func (ps *PubSub) NextMessageID() int32 {
	return atomic.AddInt32(&ps.messageID, 1)
}

func (ps *PubSub) GetOrCreateTopic(name string) *Topic {
	ps.topicsMu.RLock()
	topic, exists := ps.topics[name]
	ps.topicsMu.RUnlock()

	if exists {
		return topic
	}

	ps.topicsMu.Lock()
	defer ps.topicsMu.Unlock()
	if topic, exists := ps.topics[name]; exists {
		return topic
	}
	newTopic := NewTopic()
	ps.topics[name] = newTopic
	return newTopic
}

func (ps *PubSub) AddSubscriber(topic *Topic, msgCh chan message.Message) error {
	topic.Mu.Lock()
	defer topic.Mu.Unlock()

	if _, exists := topic.Subscribers[msgCh]; exists {
		return fmt.Errorf("subscriber already exists")
	}

	topic.Subscribers[msgCh] = struct{}{}
	return nil
}

func (ps *PubSub) RemoveSubscriber(topic *Topic, msgCh chan message.Message) error {
	topic.Mu.Lock()
	defer topic.Mu.Unlock()

	if _, exists := topic.Subscribers[msgCh]; exists {
		delete(topic.Subscribers, msgCh)
		close(msgCh)
		return nil
	}
	log.Println("Warning: Attempted to remove non-existent subscriber")
	return fmt.Errorf("subscriber does not exist")
}

func handleSlowSubscriber(topic *Topic, msg message.Message) {
	if len(topic.MessageQueue) >= MaxMessages {
		topic.MessageQueue = topic.MessageQueue[1:]
		log.Println("Dropped oldest message due to slow subscriber")
	}
	topic.MessageQueue = append(topic.MessageQueue, msg)
}
