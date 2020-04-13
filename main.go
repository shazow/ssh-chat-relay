package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Version of the binary, assigned during build.
var Version = "dev"

var logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

// Options contains the flag options
type Options struct {
	Args struct {
		Host string `positional-arg-name:"Host" description:"SSH hostname:port to connect with and relay"`
	} `positional-args:"yes"`

	Username string `long:"name" description:"Username to connect with" default:"ssh-relay"`

	Verbose []bool `long:"verbose" short:"v" description:"Show verbose logging."`
	Version bool   `long:"version" description:"Print version and exit."`
}

func exit(code int, format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(code)
}

func main() {
	options := Options{}
	p, err := flags.NewParser(&options, flags.Default).ParseArgs(os.Args[1:])
	if err != nil {
		if p == nil {
			fmt.Println(err)
		}
		return
	}

	if options.Version {
		fmt.Println(Version)
		os.Exit(0)
	}

	// Logging
	switch len(options.Verbose) {
	case 0:
		logger = logger.Level(zerolog.WarnLevel)
	case 1:
		logger = logger.Level(zerolog.InfoLevel)
	default:
		logger = logger.Level(zerolog.DebugLevel)
	}

	// Signals
	ctx, abort := context.WithCancel(context.Background())
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func(abort context.CancelFunc) {
		<-sigCh
		logger.Warn().Msg("interrupt received, shutting down")
		abort()
		<-sigCh
		logger.Error().Msg("second interrupt received, panicking")
		panic("aborted")
	}(abort)

	if err := run(ctx, options); err != nil {
		exit(1, "failed: %s\n", err)
	}
}

func run(ctx context.Context, options Options) error {
	conn := sshConnection{
		Host: options.Args.Host,
		Name: options.Username,
		Term: "bot",
	}

	logger.Info().Str("host", conn.Host).Str("name", conn.Name).Msg("connecting")

	if err := conn.Connect(); err != nil {
		return err
	}
	defer conn.Close()

	relay := ioRelay{
		RelayHandlers: RelayHandlers{
			OnMessage: func(msg string) {
				logger.Debug().Str("received", msg).Msg("msg")
			},
		},
	}

	return relay.Serve(ctx, conn.Reader, conn.Writer)
}
