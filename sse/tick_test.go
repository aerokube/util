package sse

import (
	"context"
	"encoding/json"
	. "github.com/aandryashin/matchers"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

type MockBroker struct {
	messages chan string
}

func (mb *MockBroker) HasClients() bool {
	return true
}

func (mb *MockBroker) Notify(data []byte) {
	mb.messages <- string(data)
}

func (mb *MockBroker) ServeHTTP(rw http.ResponseWriter, _ *http.Request) {
	rw.WriteHeader(http.StatusOK)
}

func TestTick(t *testing.T) {
	srv := httptest.NewServer(mockApi())
	broker := &MockBroker{messages: make(chan string, 10)}
	stop := make(chan os.Signal)
	go Tick(broker, func(ctx context.Context, br Broker) {
		req, _ := http.NewRequest("GET", srv.URL+"/status", nil)
		resp, _ := http.DefaultClient.Do(req)
		data, _ := io.ReadAll(resp.Body)
		br.Notify(data)
	}, 10*time.Millisecond, stop)
	time.Sleep(50 * time.Millisecond)
	close(stop)
	AssertThat(t, len(broker.messages) > 0, Is{true})
}

func mockApi() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/status", mockStatus)
	return mux
}

type SomeState struct {
	Field string `json:"field"`
}

func mockStatus(w http.ResponseWriter, _ *http.Request) {
	data, _ := json.MarshalIndent(SomeState{}, "", " ")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
