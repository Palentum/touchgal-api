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

func TestSMTPMailerSendsMultipartVerificationCode(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	message := make(chan string, 1)
	serverDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(3 * time.Second))

		reader := bufio.NewReader(conn)
		write := func(response string) bool {
			if _, err := conn.Write([]byte(response)); err != nil {
				serverDone <- err
				return false
			}
			return true
		}

		if !write("220 example ESMTP\r\n") {
			return
		}
		if _, err := reader.ReadString('\n'); err != nil {
			serverDone <- err
			return
		}
		if !write("250-example\r\n250 SIZE 1000\r\n") {
			return
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			serverDone <- err
			return
		}
		if !strings.HasPrefix(line, "MAIL FROM:") {
			serverDone <- errors.New("expected MAIL FROM command, got " + line)
			return
		}
		if !write("250 ok\r\n") {
			return
		}

		line, err = reader.ReadString('\n')
		if err != nil {
			serverDone <- err
			return
		}
		if !strings.HasPrefix(line, "RCPT TO:") {
			serverDone <- errors.New("expected RCPT TO command, got " + line)
			return
		}
		if !write("250 ok\r\n") {
			return
		}

		line, err = reader.ReadString('\n')
		if err != nil {
			serverDone <- err
			return
		}
		if !strings.HasPrefix(line, "DATA") {
			serverDone <- errors.New("expected DATA command, got " + line)
			return
		}
		if !write("354 end with dot\r\n") {
			return
		}

		var data strings.Builder
		for {
			line, err = reader.ReadString('\n')
			if err != nil {
				serverDone <- err
				return
			}
			if line == ".\r\n" {
				break
			}
			data.WriteString(line)
		}
		message <- data.String()
		if !write("250 queued\r\n") {
			return
		}

		line, err = reader.ReadString('\n')
		if err != nil {
			serverDone <- err
			return
		}
		if !strings.HasPrefix(line, "QUIT") {
			serverDone <- errors.New("expected QUIT command, got " + line)
			return
		}
		if !write("221 bye\r\n") {
			return
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
		SMTPFrom:               "no-reply@example.com",
		MailSendTimeoutSeconds: 1,
	})

	if err := mailer.SendVerificationCode("user@example.com", "login", "123456", 10); err != nil {
		t.Fatalf("send verification code: %v", err)
	}
	if err := <-serverDone; err != nil {
		t.Fatalf("smtp test server: %v", err)
	}

	select {
	case body := <-message:
		for _, want := range []string{
			"Content-Type: multipart/alternative;",
			"Content-Type: text/plain; charset=UTF-8",
			"Content-Type: text/html; charset=UTF-8",
			"123456",
			"#cc785c",
			"#181715",
		} {
			if !strings.Contains(body, want) {
				t.Fatalf("SMTP message missing %q:\n%s", want, body)
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SMTP test server did not capture message body")
	}
}
