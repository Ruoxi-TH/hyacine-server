package email

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
)

type SMTPConfig struct {
	Host       string
	Port       int
	User       string
	Password   string
	From       string
	Encryption string
}

type Sender struct {
	config SMTPConfig
}

func NewSender(config SMTPConfig) *Sender {
	return &Sender{config: config}
}

func (s *Sender) Send(to, subject, body string) error {
	if s.config.Host == "" {
		return fmt.Errorf("SMTP not configured")
	}

	from := s.config.From
	if from == "" {
		from = s.config.User
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-version: 1.0;\r\nContent-Type: text/plain; charset=\"UTF-8\";\r\n\r\n%s",
		from, to, subject, body)

	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	auth := smtp.PlainAuth("", s.config.User, s.config.Password, s.config.Host)

	encryption := strings.ToLower(s.config.Encryption)
	if encryption == "" {
		encryption = "starttls"
	}

	if encryption == "tls" {
		return s.sendTLS(addr, auth, from, to, msg)
	}
	return s.sendSTARTTLS(addr, auth, from, to, msg)
}

func (s *Sender) sendSTARTTLS(addr string, auth smtp.Auth, from, to, msg string) error {
	client, err := smtp.Dial(addr)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer client.Close()

	if ok, _ := client.Extension("STARTTLS"); ok {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: false,
			ServerName:         s.config.Host,
		}
		if err := client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to start TLS: %w", err)
		}
	}

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return client.Quit()
}

func (s *Sender) sendTLS(addr string, auth smtp.Auth, from, to, msg string) error {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: false,
		ServerName:         s.config.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, s.config.Host)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %w", err)
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	if err := client.Mail(from); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("failed to set recipient: %w", err)
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}

	if _, err := w.Write([]byte(msg)); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	return client.Quit()
}

func (s *Sender) SendVerificationCode(to, code string) error {
	subject := "风堇音乐 - 验证码"
	body := fmt.Sprintf("您的验证码是：%s\n\n验证码在 5 分钟内有效，请勿泄露给他人。\n\n—— 风堇音乐团队", code)
	return s.Send(to, subject, body)
}

func (s *Sender) SendBanNotification(to, reason string) error {
	subject := "风堇音乐 - 账号封禁通知"
	body := fmt.Sprintf("您的账号已被封禁。\n\n封禁原因：%s\n\n如有疑问，请联系客服。\n\n—— 风堇音乐团队", reason)
	return s.Send(to, subject, body)
}

func (s *Sender) SendUnbanNotification(to string) error {
	subject := "风堇音乐 - 账号解封通知"
	body := "您的账号已解封，现在可以正常使用。\n\n—— 风堇音乐团队"
	return s.Send(to, subject, body)
}

func FormatFrom(name, email string) string {
	if name == "" {
		return email
	}
	return fmt.Sprintf("%s <%s>", name, strings.TrimSpace(email))
}