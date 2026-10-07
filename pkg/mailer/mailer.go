// Package mailer отправляет письма через SMTP (Yandex Cloud Postbox) или в лог.
package mailer

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/smtp"
	"net/textproto"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	// From — «Имя <адрес>» или просто адрес.
	From string
}

type SMTP struct {
	cfg  Config
	from *mail.Address
}

func NewSMTP(cfg Config) (*SMTP, error) {
	from, err := mail.ParseAddress(cfg.From)
	if err != nil {
		return nil, fmt.Errorf("mailer: MAIL_FROM: %w", err)
	}
	if cfg.Host == "" || cfg.Port == 0 {
		return nil, errors.New("mailer: SMTP_HOST и SMTP_PORT обязательны")
	}
	return &SMTP{cfg: cfg, from: from}, nil
}

// Send: порт 465 — TLS сразу, иначе STARTTLS (обязателен).
func (s *SMTP) Send(ctx context.Context, to, subject, html, text string) error {
	msg, err := buildMessage(s.from, to, subject, html, text, time.Now())
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port))
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	tlsCfg := &tls.Config{ServerName: s.cfg.Host, MinVersion: tls.VersionTLS12}

	var conn net.Conn
	if s.cfg.Port == 465 {
		conn, err = (&tls.Dialer{NetDialer: dialer, Config: tlsCfg}).DialContext(ctx, "tcp", addr)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("mailer: dial: %w", err)
	}
	deadline := time.Now().Add(30 * time.Second)
	if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
		deadline = d
	}
	_ = conn.SetDeadline(deadline)
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		}
	}()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))

	c, err := smtp.NewClient(conn, s.cfg.Host)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("mailer: hello: %w", err)
	}
	defer c.Close()

	if s.cfg.Port != 465 {
		if ok, _ := c.Extension("STARTTLS"); !ok {
			return errors.New("mailer: сервер не умеет STARTTLS")
		}
		if err := c.StartTLS(tlsCfg); err != nil {
			return fmt.Errorf("mailer: starttls: %w", err)
		}
	}
	if s.cfg.User != "" {
		if err := c.Auth(smtp.PlainAuth("", s.cfg.User, s.cfg.Password, s.cfg.Host)); err != nil {
			return fmt.Errorf("mailer: auth: %w", err)
		}
	}
	if err := c.Mail(s.from.Address); err != nil {
		return fmt.Errorf("mailer: mail from: %w", err)
	}
	if err := c.Rcpt(to); err != nil {
		return fmt.Errorf("mailer: rcpt: %w", err)
	}
	w, err := c.Data()
	if err != nil {
		return fmt.Errorf("mailer: data: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("mailer: write: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mailer: close data: %w", err)
	}
	_ = c.Quit() // письмо уже принято; ошибка Quit не повод повторять отправку
	return nil
}

// buildMessage собирает multipart/alternative: текст и HTML в quoted-printable.
func buildMessage(from *mail.Address, to, subject, html, text string, now time.Time) ([]byte, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	for _, part := range []struct{ ct, content string }{
		{"text/plain; charset=UTF-8", text},
		{"text/html; charset=UTF-8", html},
	} {
		h := textproto.MIMEHeader{}
		h.Set("Content-Type", part.ct)
		h.Set("Content-Transfer-Encoding", "quoted-printable")
		w, err := mw.CreatePart(h)
		if err != nil {
			return nil, err
		}
		qp := quotedprintable.NewWriter(w)
		if _, err := qp.Write([]byte(part.content)); err != nil {
			return nil, err
		}
		if err := qp.Close(); err != nil {
			return nil, err
		}
	}
	if err := mw.Close(); err != nil {
		return nil, err
	}

	id := make([]byte, 12)
	if _, err := rand.Read(id); err != nil {
		return nil, err
	}
	domain := "localhost"
	if at := strings.LastIndex(from.Address, "@"); at >= 0 {
		domain = from.Address[at+1:]
	}

	var msg bytes.Buffer
	fmt.Fprintf(&msg, "From: %s\r\n", from.String())
	fmt.Fprintf(&msg, "To: %s\r\n", to)
	fmt.Fprintf(&msg, "Subject: %s\r\n", mime.BEncoding.Encode("UTF-8", subject))
	fmt.Fprintf(&msg, "Date: %s\r\n", now.Format(time.RFC1123Z))
	fmt.Fprintf(&msg, "Message-ID: <%s@%s>\r\n", hex.EncodeToString(id), domain)
	msg.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&msg, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", mw.Boundary())
	msg.Write(body.Bytes())
	return msg.Bytes(), nil
}

// Log — почта для разработки: письмо уходит в лог.
type Log struct{}

func NewLog() *Log { return &Log{} }

func (*Log) Send(_ context.Context, to, subject, _, text string) error {
	log.Printf("mailer(log): to=%s subject=%q\n%s", to, subject, text)
	return nil
}
