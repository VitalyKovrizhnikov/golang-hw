package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"
)

func main() {
	os.Exit(run())
}

func run() int {
	var timeout time.Duration

	flag.DurationVar(&timeout, "timeout", 10*time.Second, "connection timeout")
	flag.Parse()

	if flag.NArg() != 2 {
		_, _ = fmt.Fprintln(os.Stderr, "usage: go-telnet [--timeout=10s] host port")
		return 2
	}

	address := net.JoinHostPort(flag.Arg(0), flag.Arg(1))
	client := NewTelnetClient(address, timeout, os.Stdin, os.Stdout)
	if err := client.Connect(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "...Connection failed: %v\n", err)
		return 1
	}
	defer client.Close()

	_, _ = fmt.Fprintf(os.Stderr, "...Connected to %s\n", address)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		<-ctx.Done()
		_ = client.Close()
	}()

	go func() {
		if err := client.Send(); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "...Send failed: %v\n", err)
			_ = client.Close()
			return
		}
		_, _ = fmt.Fprintln(os.Stderr, "...EOF")
	}()

	err := client.Receive()
	switch {
	case ctx.Err() != nil:
		_, _ = fmt.Fprintln(os.Stderr, "...Interrupted")
		return 0
	case err != nil:
		_, _ = fmt.Fprintf(os.Stderr, "...Connection error: %v\n", err)
		return 1
	default:
		_, _ = fmt.Fprintln(os.Stderr, "...Connection was closed by peer")
		return 0
	}
}
