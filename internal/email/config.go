package email

import "time"

type EncryptionMode string

const (
	EncryptionTLS      EncryptionMode = "tls"
	EncryptionStartTLS EncryptionMode = "starttls"
)

type SMTPConfig struct {
	Host          string
	Port          int
	Username      string
	Password      string
	FromName      string
	Encryption    EncryptionMode
	ConnectTimeout time.Duration
	SendTimeout    time.Duration
}

func (c SMTPConfig) Defaults() SMTPConfig {
	if c.Encryption == "" {
		c.Encryption = EncryptionStartTLS
	}
	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = 10 * time.Second
	}
	if c.SendTimeout == 0 {
		c.SendTimeout = 30 * time.Second
	}
	return c
}
