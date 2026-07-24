package email

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/smtp"
	"strconv"
	"strings"
)

type SMTPError struct {
	Code    int
	Message string
}

func (e *SMTPError) Error() string {
	return fmt.Sprintf("SMTP error %d: %s", e.Code, e.Message)
}

func (e *SMTPError) IsPermanent() bool {
	return e.Code >= 500 && e.Code < 600
}

func (e *SMTPError) IsTemporary() bool {
	return e.Code >= 400 && e.Code < 500
}

type Client struct {
	config SMTPConfig
}

func NewClient(config SMTPConfig) *Client {
	return &Client{config: config.Defaults()}
}

func (c *Client) Send(msg *Message) error {
	data, err := msg.Build()
	if err != nil {
		return err
	}

	switch c.config.Encryption {
	case EncryptionTLS:
		return c.sendTLS(msg, data)
	default:
		return c.sendStartTLS(msg, data)
	}
}

func (c *Client) sendTLS(msg *Message, data []byte) error {
	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)

	tlsConfig := &tls.Config{
		ServerName: c.config.Host,
		MinVersion: tls.VersionTLS12,
	}

	auth := smtp.PlainAuth("", c.config.Username, c.config.Password, c.config.Host)

	err := sendMailWithTLS(addr, tlsConfig, auth, c.config.Username, msg.To, data)
	if err != nil && isAuthError(err) {
		auth = &loginAuth{
			username: c.config.Username,
			password: c.config.Password,
			host:     c.config.Host,
		}
		err = sendMailWithTLS(addr, tlsConfig, auth, c.config.Username, msg.To, data)
	}
	return parseSMTPError(err)
}

func sendMailWithTLS(addr string, tlsConfig *tls.Config, auth smtp.Auth, from, to string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS connect failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, strings.Split(addr, ":")[0])
	if err != nil {
		return fmt.Errorf("create client failed: %w", err)
	}
	defer client.Close()

	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("auth failed: %w", err)
		}
	}

	if err := client.Mail(from); err != nil {
		return err
	}
	if err := client.Rcpt(to); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = w.Write(msg)
	if err != nil {
		return err
	}
	return w.Close()
}

func (c *Client) sendStartTLS(msg *Message, data []byte) error {
	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	auth := smtp.PlainAuth("", c.config.Username, c.config.Password, c.config.Host)

	err := smtp.SendMail(addr, auth, c.config.Username, []string{msg.To}, data)
	if err != nil && isAuthError(err) {
		auth = &loginAuth{
			username: c.config.Username,
			password: c.config.Password,
			host:     c.config.Host,
		}
		err = smtp.SendMail(addr, auth, c.config.Username, []string{msg.To}, data)
	}
	return parseSMTPError(err)
}

func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "535") || strings.Contains(msg, "authentication") || strings.Contains(msg, "5.7.8")
}

func parseSMTPError(err error) error {
	if err == nil {
		return nil
	}

	msg := err.Error()
	for i := 0; i < len(msg)-2; i++ {
		if msg[i] >= '4' && msg[i] <= '5' && msg[i+1] >= '0' && msg[i+2] >= '0' {
			if i > 0 && (msg[i-1] >= '0' && msg[i-1] <= '9') {
				continue
			}
			code, _ := strconv.Atoi(msg[i : i+3])
			return &SMTPError{
				Code:    code,
				Message: msg,
			}
		}
	}

	return err
}

type loginAuth struct {
	username, password, host string
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte{}, nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if !more {
		return nil, nil
	}

	prompt := strings.ToLower(string(fromServer))
	if strings.Contains(prompt, "user") || strings.Contains(prompt, "username") || strings.Contains(prompt, "email") {
		return []byte(base64.StdEncoding.EncodeToString([]byte(a.username))), nil
	}
	if strings.Contains(prompt, "pass") || strings.Contains(prompt, "password") {
		return []byte(base64.StdEncoding.EncodeToString([]byte(a.password))), nil
	}

	return []byte(base64.StdEncoding.EncodeToString([]byte(a.username))), nil
}