package main

import (
	"context"
	"fmt"
	"os"
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
	ctx := context.Background()

	if err := run(ctx, nil); err != nil {
		exit(1, "failed: %w\n", err)
	}
}

func run(ctx context.Context, opts *Options) error {
	host := "localhost:2022"
	name := "ssh-chat-relay"

	conn := sshConnection{
		Host: host,
		Name: name,
		Term: "bot",
	}

	if err := conn.Connect(); err != nil {
		return err
	}
	defer conn.Close()

	relay := ioRelay{
		RelayHandlers: RelayHandlers{
			OnMessage: func(msg string) {
				fmt.Println("got msg:", msg)
			},
		},
	}

	return relay.Serve(ctx, conn.Reader, conn.Writer)
}
