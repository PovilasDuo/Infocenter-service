package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Message struct {
	ID   int
	Data string
}

type Topic struct {
	Subscribers map[chan Message]struct{}
	Messages    []Message
	Mu          sync.RWMutex
}

var (
	topics    = make(map[string]*Topic)
	topicsMu  sync.RWMutex
	messageID int
)

func getOrCreateTopic(name string) *Topic {
	topicsMu.Lock()
	defer topicsMu.Unlock()

	topic, exists := topics[name]
	if !exists {
		topic = &Topic{
			Subscribers: make(map[chan Message]struct{}),
			Messages:    []Message{},
		}
		topics[name] = topic
	}
	return topic
}

func broadcastMessage(topic *Topic, msg Message) {
	topic.Mu.RLock()
	defer topic.Mu.RUnlock()

	for ch := range topic.Subscribers {
		select {
		case ch <- msg:
		default:
		}
	}
}

func nextMessageID() int {
	topicsMu.Lock()
	defer topicsMu.Unlock()
	messageID++
	return messageID
}

func handleSendMessage(w http.ResponseWriter, r *http.Request) {
	topicName := r.URL.Path[len("/infocenter/"):]
	topic := getOrCreateTopic(topicName)

	messageData := make([]byte, r.ContentLength)
	r.Body.Read(messageData)

	message := Message{
		ID:   nextMessageID(),
		Data: string(messageData),
	}

	topic.Mu.Lock()
	topic.Messages = append(topic.Messages, message)
	topic.Mu.Unlock()

	broadcastMessage(topic, message)
	w.WriteHeader(http.StatusNoContent)
}

func handleReceiveMessages(w http.ResponseWriter, r *http.Request) {
	topicName := r.URL.Path[len("/infocenter/"):]
	topic := getOrCreateTopic(topicName)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	msgCh := make(chan Message)
	topic.Mu.Lock()
	topic.Subscribers[msgCh] = struct{}{}
	topic.Mu.Unlock()

	connectedAt := time.Now()
	timeout := time.After(30 * time.Second)

	for {
		select {
		case msg := <-msgCh:
			fmt.Fprintf(w, "id: %d\n", msg.ID)
			fmt.Fprintf(w, "event: msg\n")
			fmt.Fprintf(w, "data: %s\n\n", msg.Data)
			flusher.Flush()
		case <-timeout:
			duration := time.Since(connectedAt).Seconds()
			fmt.Fprintf(w, "event: timeout\n")
			fmt.Fprintf(w, "data: %.0fs\n\n", duration)
			flusher.Flush()
			return
		case <-r.Context().Done():
			return
		}
	}
}

func main() {
	http.HandleFunc("/infocenter/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			handleSendMessage(w, r)
		} else if r.Method == http.MethodGet {
			handleReceiveMessages(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	log.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}