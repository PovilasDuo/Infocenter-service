package pubsub

import (
	"errors"
	"log"

	"github.com/PovilasDuo/Infocenter-service/message"
)

func BroadcastMessage(topic *Topic, msg message.Message) error {
	topic.Mu.Lock()
	defer topic.Mu.Unlock()

	for ch := range topic.Subscribers {
		select {
		case ch <- msg:
		default:
			if len(topic.MessageQueue) < MaxMessages {
				topic.MessageQueue = append(topic.MessageQueue, msg)
			} else {
				log.Println("MessageQueue at max capacity, message discarded")
				return errors.New("MessageQueue at max capacity, message discarded")
			}
		}
	}
	return nil
}
