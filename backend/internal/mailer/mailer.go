// Package mailer implements auth.Mailer over net/smtp: it composes the
// magic-link and invite emails and delivers them through a configured SMTP
// relay. It is a leaf adapter — package main maps config.SMTPConfig onto
// mailer.Config and injects the result into the auth service.
//
// The MIME body is hand-assembled (a multipart/alternative of text/plain and
// text/html). If an operator's provider ever needs implicit TLS quirks,
// XOAUTH2, or richer MIME, swapping in a third-party mail library is contained
// to this package because auth depends only on the interface.
package mailer

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net"
	"net/smtp"
	"strings"
	"time"
)

// TLSMode selects the transport security for the SMTP connection.
type TLSMode string

const (
	// TLSStartTLS dials plaintext then upgrades with STARTTLS (port 587).
	TLSStartTLS TLSMode = "starttls"
	// TLSImplicit dials straight into TLS (port 465).
	TLSImplicit TLSMode = "implicit"
	// TLSNone dials plaintext and stays there (local dev only).
	TLSNone TLSMode = "none"
)

// Config is the SMTP connection configuration.
type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	TLS      TLSMode
}

// Mailer sends transactional email over SMTP. It satisfies auth.Mailer.
type Mailer struct {
	cfg    Config
	dialer *net.Dialer
	// now is overridable in tests for a deterministic Date header.
	now func() time.Time
}

// New returns a Mailer for cfg.
func New(cfg Config) *Mailer {
	if cfg.TLS == "" {
		cfg.TLS = TLSStartTLS
	}
	return &Mailer{cfg: cfg, dialer: &net.Dialer{Timeout: 10 * time.Second}, now: time.Now}
}

// SendMagicLink emails a sign-in link.
func (m *Mailer) SendMagicLink(ctx context.Context, addr, link string) error {
	text := "Sign in to Family Finances by opening this link:\n\n" + link +
		"\n\nThe link is single-use and expires shortly. If you did not request " +
		"it, you can ignore this email."
	html := paragraphs(
		`Sign in to Family Finances by opening this link:`,
		`<a href="`+htmlEscape(link)+`">`+htmlEscape(link)+`</a>`,
		`The link is single-use and expires shortly. If you did not request it, you can ignore this email.`,
	)
	return m.send(ctx, addr, "Your Family Finances sign-in link", text, html)
}

// SendInvite emails an acceptance link, naming the inviter.
func (m *Mailer) SendInvite(ctx context.Context, addr, link, invitedByEmail string) error {
	by := invitedByEmail
	if by == "" {
		by = "a member"
	}
	text := by + " invited you to Family Finances. Accept the invitation by opening this link:\n\n" +
		link + "\n\nThe link is single-use and expires after a while."
	html := paragraphs(
		htmlEscape(by)+` invited you to Family Finances. Accept the invitation by opening this link:`,
		`<a href="`+htmlEscape(link)+`">`+htmlEscape(link)+`</a>`,
		`The link is single-use and expires after a while.`,
	)
	return m.send(ctx, addr, "You're invited to Family Finances", text, html)
}

// send composes the MIME message and delivers it.
func (m *Mailer) send(ctx context.Context, to, subject, textBody, htmlBody string) error {
	msg, err := m.compose(to, subject, textBody, htmlBody)
	if err != nil {
		return err
	}

	client, err := m.connect(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	if m.cfg.Username != "" {
		if ok, _ := client.Extension("AUTH"); ok {
			if err := client.Auth(smtp.PlainAuth("", m.cfg.Username, m.cfg.Password, m.cfg.Host)); err != nil {
				return fmt.Errorf("smtp auth: %w", err)
			}
		}
	}

	from, err := parseAddress(m.cfg.From)
	if err != nil {
		return err
	}
	if err := client.Mail(from); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// connect dials the relay and, for STARTTLS, upgrades the connection.
func (m *Mailer) connect(ctx context.Context) (*smtp.Client, error) {
	address := net.JoinHostPort(m.cfg.Host, m.cfg.Port)
	tlsCfg := &tls.Config{ServerName: m.cfg.Host}

	var conn net.Conn
	var err error
	if m.cfg.TLS == TLSImplicit {
		conn, err = tls.DialWithDialer(m.dialer, "tcp", address, tlsCfg)
	} else {
		conn, err = m.dialer.DialContext(ctx, "tcp", address)
	}
	if err != nil {
		return nil, fmt.Errorf("smtp dial %s: %w", address, err)
	}

	client, err := smtp.NewClient(conn, m.cfg.Host)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("smtp handshake: %w", err)
	}

	if m.cfg.TLS == TLSStartTLS {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(tlsCfg); err != nil {
				client.Close()
				return nil, fmt.Errorf("smtp STARTTLS: %w", err)
			}
		}
	}
	return client, nil
}

func (m *Mailer) compose(to, subject, textBody, htmlBody string) ([]byte, error) {
	from, err := parseAddress(m.cfg.From)
	if err != nil {
		return nil, err
	}
	boundary := randomBoundary()

	var b bytes.Buffer
	fmt.Fprintf(&b, "From: %s\r\n", m.cfg.From)
	fmt.Fprintf(&b, "To: %s\r\n", to)
	fmt.Fprintf(&b, "Subject: %s\r\n", subject)
	fmt.Fprintf(&b, "Date: %s\r\n", m.now().Format(time.RFC1123Z))
	fmt.Fprintf(&b, "Message-ID: <%s@%s>\r\n", randomToken(), domainOf(from))
	b.WriteString("MIME-Version: 1.0\r\n")
	fmt.Fprintf(&b, "Content-Type: multipart/alternative; boundary=%q\r\n\r\n", boundary)

	fmt.Fprintf(&b, "--%s\r\n", boundary)
	b.WriteString("Content-Type: text/plain; charset=utf-8\r\n\r\n")
	b.WriteString(normalizeNewlines(textBody))
	b.WriteString("\r\n")

	fmt.Fprintf(&b, "--%s\r\n", boundary)
	b.WriteString("Content-Type: text/html; charset=utf-8\r\n\r\n")
	b.WriteString(normalizeNewlines(htmlBody))
	b.WriteString("\r\n")

	fmt.Fprintf(&b, "--%s--\r\n", boundary)
	return b.Bytes(), nil
}

// --- small helpers ----------------------------------------------------

func parseAddress(s string) (string, error) {
	s = strings.TrimSpace(s)
	if i := strings.LastIndexByte(s, '<'); i >= 0 {
		if j := strings.IndexByte(s[i:], '>'); j >= 0 {
			s = s[i+1 : i+j]
		}
	}
	if s == "" || !strings.Contains(s, "@") {
		return "", fmt.Errorf("mailer: invalid SMTP_FROM %q", s)
	}
	return s, nil
}

func domainOf(addr string) string {
	if i := strings.LastIndexByte(addr, '@'); i >= 0 {
		return addr[i+1:]
	}
	return "localhost"
}

func normalizeNewlines(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	return strings.ReplaceAll(s, "\n", "\r\n")
}

func htmlEscape(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}

func paragraphs(ps ...string) string {
	var b strings.Builder
	b.WriteString("<html><body>")
	for _, p := range ps {
		b.WriteString("<p>")
		b.WriteString(p)
		b.WriteString("</p>")
	}
	b.WriteString("</body></html>")
	return b.String()
}

func randomBoundary() string { return "ff-" + randomToken() }

func randomToken() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
