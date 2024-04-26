package main

import (
	"bytes"
	"flag"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"strings"
	"text/template"
)

func main() {
	addr := flag.String("addr", os.Getenv("MAIL_SENDER_ADDR"), "addr")
	user := flag.String("user", os.Getenv("MAIL_SENDER_USER"), "user")
	pass := flag.String("pass", os.Getenv("MAIL_SENDER_PASS"), "pass")
	to := flag.String("to", os.Getenv("MAIL_SENDER_TO"), "to")
	from := flag.String("from", os.Getenv("MAIL_SENDER_FROM"), "from")
	subject := flag.String("subject", "", "subject")
	body := flag.String("body", "", "body")
	flag.Parse()
	sender := Sender{Addr: *addr, Password: *pass, Username: *user}
	message := Message{From: *from, To: *to, Subject: *subject, Body: *body}
	if err := sender.Send(message); err != nil {
		slog.Error("send fail.", "err", err)
		os.Exit(1)
	}
	slog.Info("send success.")
}

type Message struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type Sender struct {
	Addr     string `json:"addr"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Sender) Send(message Message) error {
	if s.Addr == "" {
		return fmt.Errorf("no addr")
	}
	if message.To == "" {
		return fmt.Errorf("no to")
	}

	if message.From != "" {
		address, err := mail.ParseAddress(message.From)
		if err != nil {
			return err
		}
		if address.Name != "" {
			message.From = fmt.Sprintf("%s <%s>", mime.BEncoding.Encode("UTF-8", address.Name), address.Address)
		} else if address.Address != "" {
			message.From = address.Address
		}
	}
	if message.Subject != "" {
		message.Subject = mime.BEncoding.Encode("UTF-8", message.Subject)
	}
	if message.Body != "" {
		message.Body = strings.NewReplacer("\\r", "\r", "\\n", "\n").Replace(message.Body)
	}

	tpl, err := template.New("message").Parse("From: {{.From}}\r\nTo: {{.To}}\r\nSubject: {{.Subject}}\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n{{.Body}}")
	if err != nil {
		return err
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, message); err != nil {
		return err
	}

	host, _, err := net.SplitHostPort(s.Addr)
	if err != nil {
		return err
	}
	auth := smtp.PlainAuth("", s.Username, s.Password, host)
	if err := smtp.SendMail(s.Addr, auth, s.Username, strings.Split(message.To, ","), buf.Bytes()); err != nil {
		return err
	}
	return nil
}
