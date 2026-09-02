package mailer

import (
	"bufio"
	"context"
	"net"
	"strings"
	"sync"
	"testing"
	"time"
)

// captureSMTP is a minimal in-process SMTP server that accepts one message and
// records the envelope and DATA payload. Enough of RFC 5321 to satisfy
// net/smtp's client with no TLS and no AUTH.
type captureSMTP struct {
	ln   net.Listener
	mu   sync.Mutex
	from string
	to   []string
	data string
	done chan struct{}
}

func startCapture(t *testing.T) *captureSMTP {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	c := &captureSMTP{ln: ln, done: make(chan struct{})}
	go c.serve()
	t.Cleanup(func() { ln.Close() })
	return c
}

func (c *captureSMTP) addr() (host, port string) {
	h, p, _ := net.SplitHostPort(c.ln.Addr().String())
	return h, p
}

func (c *captureSMTP) serve() {
	conn, err := c.ln.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	defer close(c.done)

	r := bufio.NewReader(conn)
	w := conn
	write := func(s string) { _, _ = w.Write([]byte(s + "\r\n")) }

	write("220 capture ESMTP")
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		cmd := strings.ToUpper(strings.TrimSpace(line))
		switch {
		case strings.HasPrefix(cmd, "EHLO"), strings.HasPrefix(cmd, "HELO"):
			write("250-capture")
			write("250 OK")
		case strings.HasPrefix(cmd, "MAIL FROM:"):
			c.mu.Lock()
			c.from = strings.TrimSpace(line[len("MAIL FROM:"):])
			c.mu.Unlock()
			write("250 OK")
		case strings.HasPrefix(cmd, "RCPT TO:"):
			c.mu.Lock()
			c.to = append(c.to, strings.TrimSpace(line[len("RCPT TO:"):]))
			c.mu.Unlock()
			write("250 OK")
		case cmd == "DATA":
			write("354 End data with <CR><LF>.<CR><LF>")
			var b strings.Builder
			for {
				dl, err := r.ReadString('\n')
				if err != nil {
					return
				}
				if dl == ".\r\n" || dl == ".\n" {
					break
				}
				b.WriteString(dl)
			}
			c.mu.Lock()
			c.data = b.String()
			c.mu.Unlock()
			write("250 OK queued")
		case cmd == "QUIT":
			write("221 Bye")
			return
		default:
			write("250 OK")
		}
	}
}

func (c *captureSMTP) payload(t *testing.T) (from string, to []string, data string) {
	t.Helper()
	select {
	case <-c.done:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for the message")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.from, append([]string(nil), c.to...), c.data
}

func TestSendMagicLinkOverSMTP(t *testing.T) {
	cap := startCapture(t)
	host, port := cap.addr()

	m := New(Config{Host: host, Port: port, From: "Family Finances <no-reply@ff.example>", TLS: TLSNone})
	link := "https://ff.example/api/auth/email/callback?token=abc123"
	if err := m.SendMagicLink(context.Background(), "user@example.com", link); err != nil {
		t.Fatalf("SendMagicLink: %v", err)
	}

	from, to, data := cap.payload(t)
	if !strings.Contains(from, "no-reply@ff.example") {
		t.Errorf("MAIL FROM = %q", from)
	}
	if len(to) != 1 || !strings.Contains(to[0], "user@example.com") {
		t.Errorf("RCPT TO = %v", to)
	}
	if !strings.Contains(data, "To: user@example.com") {
		t.Errorf("missing To header:\n%s", data)
	}
	if !strings.Contains(data, "Subject: Your Family Finances sign-in link") {
		t.Errorf("missing/instant Subject header:\n%s", data)
	}
	if !strings.Contains(data, "Content-Type: multipart/alternative") {
		t.Errorf("not multipart/alternative:\n%s", data)
	}
	if !strings.Contains(data, "text/plain") || !strings.Contains(data, "text/html") {
		t.Errorf("missing a MIME part:\n%s", data)
	}
	if strings.Count(data, link) < 2 {
		t.Errorf("link not present in both parts:\n%s", data)
	}
}

func TestSendInviteOverSMTP(t *testing.T) {
	cap := startCapture(t)
	host, port := cap.addr()

	m := New(Config{Host: host, Port: port, From: "no-reply@ff.example", TLS: TLSNone})
	link := "https://ff.example/api/auth/invites/accept?token=xyz789"
	if err := m.SendInvite(context.Background(), "invitee@example.com", link, "host@ff.example"); err != nil {
		t.Fatalf("SendInvite: %v", err)
	}

	_, to, data := cap.payload(t)
	if len(to) != 1 || !strings.Contains(to[0], "invitee@example.com") {
		t.Errorf("RCPT TO = %v", to)
	}
	if !strings.Contains(data, "Subject: You're invited to Family Finances") {
		t.Errorf("missing Subject:\n%s", data)
	}
	if !strings.Contains(data, "host@ff.example") {
		t.Errorf("inviter not named in body:\n%s", data)
	}
	if !strings.Contains(data, link) {
		t.Errorf("acceptance link missing:\n%s", data)
	}
}
