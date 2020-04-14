package main

import (
	"context"
	"io"

	"golang.org/x/crypto/ssh"
)

// newClientConfig creates a barebones ssh.ClientConfig to be used with ssh.Dial.
func newClientConfig(name string) *ssh.ClientConfig {
	return &ssh.ClientConfig{
		User: name,
		Auth: []ssh.AuthMethod{
			ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
				return
			}),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}

type sshConnection struct {
	Addr string // Addr is the host:port to connect to on Connect()
	Name string // Name is the client username to connect with

	Term         string // TERM env var to send, suggested: "bot" or "xterm"
	ClientConfig *ssh.ClientConfig

	Reader io.Reader
	Writer io.WriteCloser

	conn    *ssh.Client
	session *ssh.Session
}

func (sshConn *sshConnection) Close() error {
	if sshConn.conn != nil {
		sshConn.conn.Close()
	}
	if sshConn.session != nil {
		sshConn.session.Close()
	}
	return nil
}

func (sshConn *sshConnection) Connect(ctx context.Context) error {
	config := sshConn.ClientConfig
	if config == nil {
		config = newClientConfig(sshConn.Name)
	}
	conn, err := ssh.Dial("tcp", sshConn.Addr, config)
	if err != nil {
		return err
	}

	go func() {
		<-ctx.Done()
		logger.Debug().Msg("ssh connection aborted")
		sshConn.Close()
	}()

	session, err := conn.NewSession()
	if err != nil {
		return err
	}

	if sshConn.Writer, err = session.StdinPipe(); err != nil {
		return err
	}

	if sshConn.Reader, err = session.StdoutPipe(); err != nil {
		return err
	}

	if err := session.RequestPty(sshConn.Term, 1000, 100, ssh.TerminalModes{}); err != nil {
		return err
	}

	if err := session.Shell(); err != nil {
		return err
	}

	if _, err = session.SendRequest("ping", true, nil); err != nil {
		return err
	}

	return nil
}
