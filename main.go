package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"golang.org/x/sync/errgroup"
)

// Version of the binary, assigned during build.
var Version = "dev"

// Options contains the flag options
type Options struct {
}

func exit(code int, format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(code)
}

func main() {
	host := "localhost:2022"
	name := "ssh-chat-relay"

	conn := sshConnection{
		Host: host,
		Name: name,
	}

	if err := conn.Connect(); err != nil {
		exit(1, "Connect failed: %w\n", err)
	}
	defer conn.Close()

	ctx := context.Background()

	relay := ioRelay{
		RelayHandlers: RelayHandlers{
			OnMessage: func(msg string) {
				fmt.Println("got msg:", msg)
			},
		},
	}

	if err := relay.Serve(ctx, conn.Reader, conn.Writer); err != nil {
		exit(2, "Relay serve failed: %w\n", err)
	}
}

type RelayHandlers struct {
	OnMessage func(string)
}

type Relay interface {
	OnMessage(string)
	Send(string) error
	Close() error
}

type ioRelay struct {
	RelayHandlers

	sendCh    chan string
	closeOnce sync.Once
	closeCh   chan struct{}
}

func (relay *ioRelay) init() {
	relay.sendCh = make(chan string)
	relay.closeOnce = sync.Once{}
	relay.closeCh = make(chan struct{})
}

func (relay *ioRelay) Serve(ctx context.Context, r io.Reader, w io.WriteCloser) error {
	relay.init()

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			if relay.OnMessage != nil {
				relay.OnMessage(scanner.Text())
			}
		}
		return scanner.Err()
	})
	g.Go(func() error {
		for {
			select {
			case msg := <-relay.sendCh:
				_, err := io.WriteString(w, msg)
				if err != nil {
					return err
				}
			case <-ctx.Done():
				return w.Close()
			case <-relay.closeCh:
				return w.Close()
			}
		}
		return nil
	})
	return g.Wait()
}

func (relay *ioRelay) Send(msg string) error {
	if relay.sendCh == nil {
		return errors.New("relay not initialized")
	}
	relay.sendCh <- msg
	return nil
}

func (relay *ioRelay) Close() error {
	relay.closeOnce.Do(func() {
		relay.closeCh <- struct{}{}
	})
	return nil
}
