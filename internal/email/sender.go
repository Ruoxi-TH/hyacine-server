package email

import (
	"fmt"
)

type Sender struct {
	client *Client
	config SMTPConfig
}

func NewSender(config SMTPConfig) *Sender {
	return &Sender{
		client: NewClient(config),
		config: config,
	}
}

func (s *Sender) Send(to, subject, body string) error {
	if s.config.Host == "" {
		return fmt.Errorf("SMTP not configured")
	}

	msg := NewMessage(s.config.Username, s.config.FromName, to, subject, body)
	return s.client.Send(msg)
}

func (s *Sender) SendHTML(to, subject, textBody, htmlBody string) error {
	if s.config.Host == "" {
		return fmt.Errorf("SMTP not configured")
	}

	msg := NewMessage(s.config.Username, s.config.FromName, to, subject, textBody)
	msg.SetHTML(htmlBody)
	return s.client.Send(msg)
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