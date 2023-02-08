package sse

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// the amount of time to wait when pushing a message to
// a slow client or a client that closed after `range clients` started.
const patience = 1 * time.Second

type Broker interface {
	http.Handler
	Notify(data []byte)
	HasClients() bool
}

type SseBroker struct {
	// Events are pushed to this channel
	notifier chan []byte

	// New client connections
	newClients chan chan []byte

	// Closed client connections
	closingClients chan chan []byte

	// Client connections registry
	clients map[chan []byte]bool

	lock sync.RWMutex
}

func NewSseBroker() (broker *SseBroker) {
	broker = &SseBroker{
		notifier:       make(chan []byte, 1),
		newClients:     make(chan chan []byte),
		closingClients: make(chan chan []byte),
		clients:        make(map[chan []byte]bool),
	}

	go broker.listen()

	return broker
}

func (sse *SseBroker) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	flusher, ok := rw.(http.Flusher)
	if !ok {
		http.Error(rw, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	rw.Header().Set("Content-Type", "text/event-stream")
	rw.Header().Set("Cache-Control", "no-cache")
	rw.Header().Set("Connection", "keep-alive")
	rw.Header().Set("Access-Control-Allow-Origin", "*")

	messageChan := make(chan []byte)
	sse.newClients <- messageChan
	defer func() {
		sse.closingClients <- messageChan
	}()

	for {
		select {
		case <-req.Context().Done():
			{
				return
			}
		default:
			{
				fmt.Fprintf(rw, "data: %s\n\n", <-messageChan)
				flusher.Flush()
			}
		}
	}

}

func (sse *SseBroker) Notify(data []byte) {
	sse.notifier <- data
}

func (sse *SseBroker) HasClients() bool {
	sse.lock.RLock()
	defer sse.lock.RUnlock()
	return len(sse.clients) > 0
}

func (sse *SseBroker) listen() {
	for {
		select {
		case s := <-sse.newClients:
			{
				sse.lock.Lock()
				sse.clients[s] = true
				sse.lock.Unlock()
				log.Printf("Client added. %d registered clients", len(sse.clients))
			}
		case s := <-sse.closingClients:
			{
				sse.lock.Lock()
				delete(sse.clients, s)
				sse.lock.Unlock()
				log.Printf("Removed client. %d registered clients", len(sse.clients))
			}
		case event := <-sse.notifier:
			{
				for clientMessageChan := range sse.clients {
					select {
					case clientMessageChan <- event:
					case <-time.After(patience):
						log.Print("Skipping slow client")
					}
				}
			}
		}
	}
}
