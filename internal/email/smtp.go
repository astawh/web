// Package email delivers plain-text notifications over TLS SMTP.
package email

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

type Config struct{ Address, Username, Password, From, Mode string }
type Sender struct {
	cfg       Config
	host      string
	tlsConfig *tls.Config
}

func New(cfg Config) (*Sender, error) {
	host, _, err := net.SplitHostPort(cfg.Address)
	if err != nil || host == "" {
		return nil, errors.New("SMTP_ADDRESS must contain host and port")
	}
	if !validAddress(cfg.From) {
		return nil, errors.New("SMTP_FROM must be a single email address")
	}
	if (cfg.Username == "") != (cfg.Password == "") {
		return nil, errors.New("SMTP username and password must be configured together")
	}
	if cfg.Mode == "" {
		cfg.Mode = "starttls"
	}
	if cfg.Mode != "starttls" && cfg.Mode != "tls" {
		return nil, errors.New("SMTP_MODE must be starttls or tls")
	}
	return &Sender{cfg: cfg, host: host, tlsConfig: &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}}, nil
}

func validAddress(value string) bool {
	a, err := mail.ParseAddress(value)
	return err == nil && a.Address == value && !strings.ContainsAny(value, "\r\n")
}

func (s *Sender) Send(ctx context.Context, to, subject, body string) error {
	if !validAddress(to) || strings.ContainsAny(subject, "\r\n") {
		return errors.New("invalid mail envelope")
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	dialer := &net.Dialer{Timeout: 20 * time.Second}
	var conn net.Conn
	var err error
	if s.cfg.Mode == "tls" {
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: s.tlsConfig}).DialContext(ctx, "tcp", s.cfg.Address)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", s.cfg.Address)
	}
	if err != nil {
		return err
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return err
	}
	defer client.Close()
	if s.cfg.Mode == "starttls" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return errors.New("SMTP server does not support STARTTLS")
		}
		if err := client.StartTLS(s.tlsConfig); err != nil {
			return err
		}
	}
	if s.cfg.Username != "" {
		if err := client.Auth(smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.host)); err != nil {
			return err
		}
	}
	if err := client.Mail(s.cfg.From); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	message := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: 8bit\r\n\r\n%s\r\n", s.cfg.From, to, mime.QEncoding.Encode("utf-8", subject), time.Now().UTC().Format(time.RFC1123Z), strings.ReplaceAll(strings.ReplaceAll(body, "\r\n", "\n"), "\n", "\r\n"))
	if _, err := w.Write([]byte(message)); err != nil {
		_ = w.Close()
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}
