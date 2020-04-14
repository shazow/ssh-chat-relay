package main

import (
	"context"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var MessageBuffer = 5
var MessageTimeout = time.Second * 2

type wsRelay struct {
	Bind string

	Send func(string) error

	upgrader websocket.Upgrader
	serveCtx context.Context

	once     sync.Once
	received chan string
	done     chan struct{}

	mu    sync.Mutex
	conns map[*websocket.Conn](chan string)
}

func (relay *wsRelay) OnMessage(msg string) {
	relay.mu.Lock()
	defer relay.mu.Unlock()

	for conn, ch := range relay.conns {
		select {
		case ch <- msg:
		default:
			logger.Error().Msg("connection message buffer is full, disconnecting")
			conn.Close()
		}
	}
}

func (relay *wsRelay) Serve(ctx context.Context) error {
	relay.serveCtx = ctx
	relay.once = sync.Once{}
	relay.received = make(chan string, MessageBuffer)
	relay.done = make(chan struct{})

	go func() {
		<-ctx.Done()
		logger.Debug().Msg("websocket relay http server aborted")
		relay.Close()
	}()

	s := &http.Server{
		Addr:    relay.Bind,
		Handler: relay,

		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,

		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}
	return s.ListenAndServe()
}

func (relay *wsRelay) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := relay.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error().Err(err).Msg("failed to upgrade websocket")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	msgCh := make(chan string, MessageBuffer)
	relay.mu.Lock()
	relay.conns[conn] = msgCh
	relay.mu.Unlock()

	defer func() {
		// Cleanup
		relay.mu.Lock()
		delete(relay.conns, conn)
		relay.mu.Unlock()
		conn.Close()
		close(msgCh)
	}()

	logger.Debug().Str("remote", r.RemoteAddr).Msg("websocket relay client joined")
	ctx := r.Context()
	for {
		select {
		case msg := <-msgCh:
			if err = conn.WriteMessage(websocket.TextMessage, []byte(msg)); err != nil {
				logger.Error().Err(err).Msg("failed to write message")
				return
			}
		case <-ctx.Done():
			return
		case <-relay.serveCtx.Done():
			logger.Debug().Str("remote", r.RemoteAddr).Msg("websocket relay connection aborted")
			return
		}
	}
}

func (relay *wsRelay) Close() error {
	relay.once.Do(func() {
		close(relay.received)
		close(relay.done)
	})
	return nil
}
