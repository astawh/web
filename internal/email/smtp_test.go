package email

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"
)

func TestTLSDelivery(t *testing.T) {
	certServer := httptest.NewTLSServer(nil)
	certificate := certServer.TLS.Certificates[0]
	certServer.Close()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	listener = tls.NewListener(listener, &tls.Config{Certificates: []tls.Certificate{certificate}, MinVersion: tls.VersionTLS12})
	defer listener.Close()
	messages := make(chan string, 1)
	failures := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			failures <- err
			return
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
		p := textproto.NewConn(conn)
		if err := p.PrintfLine("220 test SMTP"); err != nil {
			failures <- err
			return
		}
		for {
			line, err := p.ReadLine()
			if err != nil {
				failures <- err
				return
			}
			switch {
			case strings.HasPrefix(line, "EHLO "):
				err = p.PrintfLine("250 test")
			case strings.HasPrefix(line, "MAIL FROM:"):
				err = p.PrintfLine("250 ok")
			case strings.HasPrefix(line, "RCPT TO:"):
				err = p.PrintfLine("250 ok")
			case line == "DATA":
				if err = p.PrintfLine("354 send data"); err == nil {
					var data []byte
					data, err = io.ReadAll(p.DotReader())
					if err == nil {
						messages <- string(data)
						err = p.PrintfLine("250 queued")
					}
				}
			case line == "QUIT":
				_ = p.PrintfLine("221 bye")
				return
			default:
				err = p.PrintfLine("500 unexpected command")
			}
			if err != nil {
				failures <- err
				return
			}
		}
	}()
	sender, err := New(Config{Address: listener.Addr().String(), Mode: "tls", From: "portal@example.org"})
	if err != nil {
		t.Fatal(err)
	}
	cert, err := x509.ParseCertificate(certificate.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	sender.tlsConfig.RootCAs = pool
	if err := sender.Send(context.Background(), "anna@example.org", "Invitation", "Hello Anna\nInvitation link"); err != nil {
		t.Fatal(err)
	}
	select {
	case message := <-messages:
		if !strings.Contains(message, "To: anna@example.org") || !strings.Contains(message, "Invitation link") {
			t.Fatal("incorrect message")
		}
	case err := <-failures:
		t.Fatal(err)
	case <-time.After(5 * time.Second):
		t.Fatal("no SMTP message")
	}
}

func TestInvalidMailConfiguration(t *testing.T) {
	for _, cfg := range []Config{
		{Address: "smtp.example.org", From: "portal@example.org"},
		{Address: "smtp.example.org:587", From: "portal@example.org\r\nBcc: other@example.org"},
		{Address: "smtp.example.org:587", From: "portal@example.org", Username: "user"},
		{Address: "smtp.example.org:587", From: "portal@example.org", Mode: "plaintext"},
	} {
		if _, err := New(cfg); err == nil {
			t.Fatal("invalid configuration accepted")
		}
	}
	sender, err := New(Config{Address: "smtp.example.org:587", From: "portal@example.org"})
	if err != nil {
		t.Fatal(err)
	}
	if err := sender.Send(context.Background(), "anna@example.org", "Subject\r\nBcc: other@example.org", "body"); err == nil {
		t.Fatal("header injection accepted")
	}
}
