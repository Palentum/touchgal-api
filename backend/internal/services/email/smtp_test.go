package email

import (
	"bufio"
	"errors"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/touchgal/developer/backend/internal/config"
)

func TestSMTPMailerTimesOutWaitingForServerGreeting(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	serverDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()
		_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
		_, _ = conn.Read(make([]byte, 1))
		serverDone <- nil
	}()

	host, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener addr: %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("parse listener port: %v", err)
	}
	mailer := NewSMTPMailer(config.Config{
		SMTPHost:               host,
		SMTPPort:               port,
		SMTPFrom:               "no-reply@example.com",
		MailSendTimeoutSeconds: 1,
	})

	started := time.Now()
	err = mailer.SendVerificationCode("user@example.com", "login", "123456", 10)
	elapsed := time.Since(started)
	if err == nil {
		t.Fatal("expected SMTP timeout error")
	}
	var netErr net.Error
	if !errors.As(err, &netErr) || !netErr.Timeout() {
		t.Fatalf("expected timeout error, got %T: %v", err, err)
	}
	if elapsed >= 2*time.Second {
		t.Fatalf("expected mail send to honor timeout, elapsed=%s", elapsed)
	}
	if err := <-serverDone; err != nil && !errors.Is(err, net.ErrClosed) {
		t.Fatalf("smtp test server: %v", err)
	}
}

func TestSMTPMailerDoesNotSendCredentialsWhenAuthUnsupported(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	commands := make(chan string, 2)
	serverDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()
		reader := bufio.NewReader(conn)
		if _, err := conn.Write([]byte("220 example ESMTP\r\n")); err != nil {
			serverDone <- err
			return
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			serverDone <- err
			return
		}
		commands <- line
		if _, err := conn.Write([]byte("250-example\r\n250 SIZE 1000\r\n")); err != nil {
			serverDone <- err
			return
		}
		_ = conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		line, err = reader.ReadString('\n')
		if err == nil {
			commands <- line
		}
		serverDone <- nil
	}()

	host, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("split listener addr: %v", err)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatalf("parse listener port: %v", err)
	}
	mailer := NewSMTPMailer(config.Config{
		SMTPHost:               host,
		SMTPPort:               port,
		SMTPUsername:           "user",
		SMTPPassword:           "secret",
		SMTPFrom:               "no-reply@example.com",
		MailSendTimeoutSeconds: 1,
	})

	err = mailer.SendVerificationCode("user@example.com", "login", "123456", 10)
	if err == nil || !strings.Contains(err.Error(), "server doesn't support AUTH") {
		t.Fatalf("expected unsupported AUTH error, got %v", err)
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("smtp test server: %v", err)
	}
	close(commands)
	for command := range commands {
		if strings.HasPrefix(command, "AUTH ") {
			t.Fatalf("client sent credentials despite missing AUTH extension: %q", command)
		}
	}
}
