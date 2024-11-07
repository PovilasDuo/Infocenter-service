package pubsub

import (
	"github.com/PovilasDuo/Infocenter-service/message"
)

func BroadcastMessage(topic *Topic, msg message.Message) error {
	topic.Mu.RLock()
	defer topic.Mu.RUnlock()

	for ch := range topic.Subscribers {
		select {
		case ch <- msg:
		default:
			handleSlowSubscriber(topic, msg)
			if _, open := <-ch; !open {
				delete(topic.Subscribers, ch)
			}
		}
	}
	return nil
}
