package handler

import (
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/PovilasDuo/Infocenter-service/message"
	"github.com/PovilasDuo/Infocenter-service/pubsub"
)

type InfoCenterHandler struct {
	ps *pubsub.PubSub
}

func NewInfoCenterHandler(ps *pubsub.PubSub) *InfoCenterHandler {
	return &InfoCenterHandler{ps: ps}
}

func (h *InfoCenterHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	topicName := r.URL.Path
	if topicName == "" {
		http.Error(w, "Topic name required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPost:
		h.handleSendMessage(w, r, topicName)
	case http.MethodGet:
		h.handleReceiveMessages(w, r, topicName)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
func (h *InfoCenterHandler) handleSendMessage(w http.ResponseWriter, r *http.Request, topicName string) {
	if r.ContentLength == 0 {
		http.Error(w, "Message body cannot be empty", http.StatusBadRequest)
		return
	}

	topic := h.ps.GetOrCreateTopic(topicName)
	messageData, err := io.ReadAll(r.Body)
	if err != nil || len(messageData) == 0 {
		http.Error(w, "Failed to read message or message is empty", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	msg := message.Message{
		ID:   h.ps.NextMessageID(),
		Data: string(messageData),
	}

	const maxRetries = 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		topic.Mu.Lock()
		if len(topic.Messages) >= pubsub.MaxMessages {
			topic.Messages = topic.Messages[1:]
		}
		topic.Messages = append(topic.Messages, msg)
		topic.Mu.Unlock()

		if err := pubsub.BroadcastMessage(topic, msg); err == nil {
			w.WriteHeader(http.StatusNoContent)
			return
		} else {
			log.Printf("Attempt %d failed with error: %v", attempt+1, err)
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
		}
	}

	http.Error(w, "Failed to send message after multiple attempts.", http.StatusInternalServerError)
}

func (h *InfoCenterHandler) handleReceiveMessages(w http.ResponseWriter, r *http.Request, topicName string) {
	topic := h.ps.GetOrCreateTopic(topicName)

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	msgCh := make(chan message.Message)
	h.ps.AddSubscriber(topic, msgCh)
	defer h.ps.RemoveSubscriber(topic, msgCh)

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
		case <-r.Context().Done():
			return
		}
	}
}
