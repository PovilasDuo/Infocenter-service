package handler

import (
	"net/http"
	"strings"

	"github.com/PovilasDuo/Infocenter-service/pubsub"
)

func InfoCenterHandler(w http.ResponseWriter, r *http.Request) {
	topicName := strings.TrimPrefix(r.URL.Path, "/infocenter/")
	if topicName == "" {
		http.Error(w, "Topic name required", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPost:
		if r.ContentLength == 0 {
			http.Error(w, "Message body cannot be empty", http.StatusBadRequest)
			return
		}
		pubsub.HandleSendMessage(w, r, topicName)
	case http.MethodGet:
		pubsub.HandleReceiveMessages(w, r, topicName)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
