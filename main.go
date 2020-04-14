package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"

	"net/http"
	_ "net/http/pprof"
)

// Version of the binary, assigned during build.
var Version = "dev"

var logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

// Options contains the flag options
type Options struct {
	Args struct {
		Addr string `positional-arg-name:"Addr" description:"SSH host:port to connect with and relay"`
	} `positional-args:"yes"`

	Websocket string `long:"websocket" description:"Websocket host:port to bind to and supply a relay"`
	Username  string `long:"name" description:"Username to connect with" default:"ssh-chat-relay"`
	Verbose   []bool `long:"verbose" short:"v" description:"Show verbose logging."`
	Version   bool   `long:"version" description:"Print version and exit."`

	Pprof string `long:"pprof" description:"Bind pprof on http server on this addr. (Example: \"localhost:6060\")"`
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

	if options.Pprof != "" {
		go func() {
			logger.Debug().Str("bind", options.Pprof).Msg("serving pprof http server")
			fmt.Println(http.ListenAndServe(options.Pprof, nil))
		}()
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
		Addr: options.Args.Addr,
		Name: options.Username,
		Term: "bot",
	}

	logger.Info().Str("addr", conn.Addr).Str("name", conn.Name).Msg("connecting")

	if err := conn.Connect(ctx); err != nil {
		return err
	}
	defer conn.Close()

	src := ioSource{
		RelayHandlers: RelayHandlers{
			OnMessage: func(msg string) {
				logger.Debug().Str("received", msg).Msg("msg")
			},
		},
	}

	g, ctx := errgroup.WithContext(ctx)
	if options.Websocket != "" {
		ws := wsRelay{
			Bind: options.Websocket,
			Send: src.Send,
		}
		logger.Info().Str("addr", ws.Bind).Msg("serving websocket relay")
		g.Go(func() error {
			return ws.Serve(ctx)
		})
	}

	g.Go(func() error {
		return src.Serve(ctx, conn.Reader, conn.Writer)
	})

	return g.Wait()
}
