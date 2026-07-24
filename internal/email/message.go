package email

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"net/mail"
	"strings"
	"time"
)

type Message struct {
	From    string
	FromName string
	To      string
	Subject string
	Body    string
	HTML    string
}

func NewMessage(from, fromName, to, subject, body string) *Message {
	return &Message{
		From:     from,
		FromName: fromName,
		To:       to,
		Subject:  subject,
		Body:     body,
	}
}

func (m *Message) SetHTML(html string) {
	m.HTML = html
}

func (m *Message) Build() ([]byte, error) {
	var buf bytes.Buffer

	from := m.formatAddress(m.From, m.FromName)
	fmt.Fprintf(&buf, "From: %s\r\n", from)
	fmt.Fprintf(&buf, "To: %s\r\n", m.To)
	fmt.Fprintf(&buf, "Subject: %s\r\n", encodeHeader(m.Subject))
	fmt.Fprintf(&buf, "Date: %s\r\n", time.Now().Format(time.RFC1123Z))
	fmt.Fprintf(&buf, "Message-ID: %s\r\n", generateMessageID(m.From))

	if m.HTML != "" {
		fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
		fmt.Fprintf(&buf, "Content-Type: multipart/alternative; boundary=\"%s\"\r\n", boundary)
		fmt.Fprintf(&buf, "\r\n")

		fmt.Fprintf(&buf, "--%s\r\n", boundary)
		fmt.Fprintf(&buf, "Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		fmt.Fprintf(&buf, "Content-Transfer-Encoding: base64\r\n")
		fmt.Fprintf(&buf, "\r\n")
		fmt.Fprintf(&buf, "%s\r\n", base64.StdEncoding.EncodeToString([]byte(m.Body)))

		fmt.Fprintf(&buf, "--%s\r\n", boundary)
		fmt.Fprintf(&buf, "Content-Type: text/html; charset=\"UTF-8\"\r\n")
		fmt.Fprintf(&buf, "Content-Transfer-Encoding: base64\r\n")
		fmt.Fprintf(&buf, "\r\n")
		fmt.Fprintf(&buf, "%s\r\n", base64.StdEncoding.EncodeToString([]byte(m.HTML)))

		fmt.Fprintf(&buf, "--%s--\r\n", boundary)
	} else {
		fmt.Fprintf(&buf, "MIME-Version: 1.0\r\n")
		fmt.Fprintf(&buf, "Content-Type: text/plain; charset=\"UTF-8\"\r\n")
		fmt.Fprintf(&buf, "Content-Transfer-Encoding: base64\r\n")
		fmt.Fprintf(&buf, "\r\n")
		fmt.Fprintf(&buf, "%s\r\n", base64.StdEncoding.EncodeToString([]byte(m.Body)))
	}

	return buf.Bytes(), nil
}

func (m *Message) formatAddress(email, name string) string {
	if name == "" {
		return email
	}
	return fmt.Sprintf("%s <%s>", encodeHeader(name), email)
}

func encodeHeader(s string) string {
	if isASCII(s) {
		return s
	}
	return mime.BEncoding.Encode("UTF-8", s)
}

func isASCII(s string) bool {
	for _, r := range s {
		if r > 127 {
			return false
		}
	}
	return true
}

func generateMessageID(from string) string {
	addr, err := mail.ParseAddress(from)
	if err != nil {
		addr = &mail.Address{Address: from}
	}
	domain := addr.Address
	if idx := strings.LastIndex(domain, "@"); idx >= 0 {
		domain = domain[idx+1:]
	} else {
		domain = "localhost"
	}

	var b [16]byte
	io.ReadFull(rand.Reader, b[:])

	return fmt.Sprintf("<%d.%x@%s>", time.Now().UnixNano(), b[:], domain)
}

const boundary = "hyacine_mail_boundary"