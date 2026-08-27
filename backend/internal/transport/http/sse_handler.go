package http

import (
	"fmt"
	"net/http"

	"github.com/apisentinel/apisentinel/internal/valkey"
	"github.com/go-chi/chi/v5"
)

type SSEHandler struct {
	valkeyClient *valkey.Client
}

func NewSSEHandler(valkeyClient *valkey.Client) *SSEHandler {
	return &SSEHandler{valkeyClient: valkeyClient}
}

func (h *SSEHandler) Stream(w http.ResponseWriter, r *http.Request) {
	projectId := chi.URLParam(r, "projectId")
	if projectId == "" {
		http.Error(w, "projectId is required", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Send initial connection packet
	fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\",\"projectId\":\"%s\"}\n\n", projectId)
	flusher.Flush()

	if h.valkeyClient == nil {
		return
	}

	pubsub := h.valkeyClient.Subscribe(r.Context(), "channel:events:"+projectId)
	defer pubsub.Close()

	ch := pubsub.Channel()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			fmt.Fprintf(w, "event: request.created\ndata: %s\n\n", msg.Payload)
			flusher.Flush()
		}
	}
}
