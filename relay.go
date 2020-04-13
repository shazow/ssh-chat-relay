package main

import (
	"bufio"
	"context"
	"errors"
	"io"
	"sync"

	"golang.org/x/sync/errgroup"
)

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
