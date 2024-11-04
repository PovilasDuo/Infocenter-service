package pubsub

import (
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/PovilasDuo/Infocenter-service/message"
)

var (
	messageID int
	messageMu sync.Mutex
)

func BroadcastMessage(topic *Topic, msg message.Message) {
	topic.Mu.RLock()
	defer topic.Mu.RUnlock()

	for ch := range topic.Subscribers {
		select {
		case ch <- msg:
		default:
			topic.MessageQueue = append(topic.MessageQueue, msg)
		}
	}
}

func NextMessageID() int {
	messageMu.Lock()
	defer messageMu.Unlock()
	messageID++
	return messageID
}

func HandleSendMessage(w http.ResponseWriter, r *http.Request, topicName string) {
	topic := GetOrCreateTopic(topicName)

	messageData, err := io.ReadAll(r.Body)
	if err != nil || len(messageData) == 0 {
		http.Error(w, "Failed to read message or message is empty", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	msg := message.Message{
		ID:   NextMessageID(),
		Data: string(messageData),
	}

	const maxRetries = 5
	var lastError error

	for attempt := 0; attempt < maxRetries; attempt++ {
		topic.Mu.Lock()
		topic.Messages = append(topic.Messages, msg)
		topic.Mu.Unlock()

		BroadcastMessage(topic, msg)

		if err == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		} else {
			lastError = err
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
		}
	}

	http.Error(w, "Failed to send message after multiple attempts: "+lastError.Error(), http.StatusInternalServerError)
}

func HandleReceiveMessages(w http.ResponseWriter, r *http.Request, topicName string) {
	topic := GetOrCreateTopic(topicName)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	msgCh := make(chan message.Message)
	topic.Mu.Lock()
	topic.Subscribers[msgCh] = struct{}{}
	topic.Mu.Unlock()

	defer func() {
		topic.Mu.Lock()
		delete(topic.Subscribers, msgCh)
		close(msgCh)
		topic.Mu.Unlock()
	}()

	for _, msg := range topic.MessageQueue {
		_, _ = w.Write([]byte("id: " + strconv.Itoa(msg.ID) + "\n"))
		_, _ = w.Write([]byte("event: msg\n"))
		_, _ = w.Write([]byte("data: " + msg.Data + "\n\n"))
		flusher.Flush()
	}

	timeout := time.NewTicker(30 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case msg := <-msgCh:
			_, _ = w.Write([]byte("id: " + strconv.Itoa(msg.ID) + "\n"))
			_, _ = w.Write([]byte("event: msg\n"))
			_, _ = w.Write([]byte("data: " + msg.Data + "\n\n"))
			flusher.Flush()
		case <-timeout.C:
			_, _ = w.Write([]byte("id: \nevent: timeout\n"))
			_, _ = w.Write([]byte("data: 30s\n\n"))
			flusher.Flush()
			return
		case <-r.Context().Done():
			return
		}
	}
}
