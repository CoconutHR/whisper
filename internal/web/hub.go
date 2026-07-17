package web

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

type socketClient struct {
	userID string
	conn   *websocket.Conn
	send   chan []byte
}

type hub struct {
	mu      sync.RWMutex
	clients map[string]map[*socketClient]struct{}
}

func newHub() *hub {
	return &hub{clients: map[string]map[*socketClient]struct{}{}}
}

func (h *hub) add(client *socketClient) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	clients := h.clients[client.userID]
	first := len(clients) == 0
	if clients == nil {
		clients = map[*socketClient]struct{}{}
		h.clients[client.userID] = clients
	}
	clients[client] = struct{}{}
	return first
}

func (h *hub) remove(client *socketClient) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	clients := h.clients[client.userID]
	if clients == nil {
		return false
	}
	if _, ok := clients[client]; ok {
		delete(clients, client)
		close(client.send)
	}
	if len(clients) == 0 {
		delete(h.clients, client.userID)
		return true
	}
	return false
}

func (h *hub) isOnline(userID string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[userID]) > 0
}

func (h *hub) onlineIDs() map[string]bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	result := make(map[string]bool, len(h.clients))
	for userID, clients := range h.clients {
		result[userID] = len(clients) > 0
	}
	return result
}

func (h *hub) sendTo(userID string, event any) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for client := range h.clients[userID] {
		select {
		case client.send <- payload:
		default:
		}
	}
}

func (h *hub) broadcast(event any) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, clients := range h.clients {
		for client := range clients {
			select {
			case client.send <- payload:
			default:
			}
		}
	}
}
