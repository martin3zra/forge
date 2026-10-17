package mailer

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestSMTPReturnsRejectedRecipient(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(2 * time.Second))
		fmt.Fprint(conn, "220 test SMTP\r\n")
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			switch {
			case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
				fmt.Fprint(conn, "250 test\r\n")
			case strings.HasPrefix(line, "MAIL FROM:"):
				fmt.Fprint(conn, "250 OK\r\n")
			case strings.HasPrefix(line, "RCPT TO:"):
				fmt.Fprint(conn, "550 rejected\r\n")
			default:
				return
			}
		}
	}()
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	m := New(Config{Driver: SMTP, Host: host, Port: port, FromAddress: "sender@example.com"}, testTemplates).To("grace@example.com", "Grace")
	if err := m.SendContext(context.Background(), fakeMailable{}); err == nil || !strings.Contains(err.Error(), "550") {
		t.Fatalf("SMTP rejection: %v", err)
	}
	<-done
}

func TestSMTPContextDeadline(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		conn.SetDeadline(time.Now().Add(time.Second))
		var b [1]byte
		conn.Read(b[:])
	}()
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	m := New(Config{Driver: SMTP, Host: host, Port: port}, testTemplates).To("grace@example.com", "Grace")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := m.SendContext(ctx, fakeMailable{}); err == nil {
		t.Fatal("SMTP ignored deadline")
	}
	<-done
}
