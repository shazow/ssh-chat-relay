package main

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/shazow/ssh-chat/sshd"
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
	name := "ssh-relay"
	err := sshd.ConnectShell(host, name, func(r io.Reader, w io.WriteCloser) error {
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
		return scanner.Err()
	})
	if err != nil {
		exit(1, "ConnectShell failed: %w\n", err)
	}
}
