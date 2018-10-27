package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/websocket"
)

func TestNewReverseProxyWebsocket(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := &websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatal(err)
			return
		}

		messageType, p, err := conn.ReadMessage()
		if err != nil {
			t.Fatal(err)
			return
		}

		if err = conn.WriteMessage(messageType, p); err != nil {
			t.Error(err)
			return
		}
	}))
	defer backend.Close()

	backendURL, _ := url.Parse(backend.URL)
	backendHostname, backendPort, _ := net.SplitHostPort(backendURL.Host)
	backendHost := net.JoinHostPort(backendHostname, backendPort)
	proxyURL, _ := url.Parse(backendURL.Scheme + "://" + backendHost + "/")

	proxyHandler := NewReverseProxy(proxyURL)
	setProxyUpstreamHostHeader(proxyHandler, proxyURL)
	frontend := httptest.NewServer(proxyHandler)
	defer frontend.Close()

	wsUrl := new(url.URL)
	*wsUrl = *proxyURL
	wsUrl.Scheme = "ws"
	conn, _, err := websocket.DefaultDialer.Dial(wsUrl.String(), nil)
	if err != nil {
		t.Fatal(err)
	}

	expected := "hello world"
	if err := conn.WriteMessage(websocket.TextMessage, []byte(expected)); err != nil {
		t.Error(err)
	}

	messageType, received, err := conn.ReadMessage()
	if err != nil {
		t.Error(err)
	}

	if messageType != websocket.TextMessage {
		t.Error("expected text message")
	}

	if string(received) != expected {
		t.Errorf("expected %s, received %s", expected, string(received))
	}
}
