package main

import (
	"io"

	"golang.org/x/crypto/ssh"
)

// Code base on github.com/shazow/ssh-chat/sshd

// NewClientConfig creates a barebones ssh.ClientConfig to be used with ssh.Dial.
func NewClientConfig(name string) *ssh.ClientConfig {
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
	Host string
	Name string

	Reader io.Reader
	Writer io.WriteCloser

	Term         string // TERM env var to send, suggested: "bot" or "xterm"
	ClientConfig *ssh.ClientConfig

	conn    *ssh.Client
	session *ssh.Session
}

func (sshConn *sshConnection) Close() error {
	if sshConn.conn == nil || sshConn.session == nil {
		return nil
	}
	sshConn.session.Close()
	sshConn.conn.Close()
	return nil
}

func (sshConn *sshConnection) Connect() error {
	config := sshConn.ClientConfig
	if config == nil {
		config = NewClientConfig(sshConn.Name)
	}
	conn, err := ssh.Dial("tcp", sshConn.Host, config)
	if err != nil {
		return err
	}

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
