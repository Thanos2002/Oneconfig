// Package health implements health check strategies (HTTP, TCP, shell command)
// with configurable timeout and retry intervals.
package health

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/Thanos2002/Oneconfig/internal/config"
	"github.com/Thanos2002/Oneconfig/internal/shell"
)

// Check performs a single health check as defined in the config.
func Check(ctx context.Context, hc config.HealthCheck) error {
	timeout, err := config.ParseDuration(hc.Timeout)
	if err != nil {
		timeout = 30 * time.Second
	}
	interval, err := config.ParseDuration(hc.Interval)
	if err != nil {
		interval = 2 * time.Second
	}

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		var checkErr error

		switch {
		case hc.URL != "":
			checkErr = checkHTTP(ctx, hc.URL)
		case hc.Port != 0:
			checkErr = checkTCP(ctx, hc.Port)
		case hc.Command != "":
			checkErr = checkCommand(ctx, hc.Command)
		default:
			return fmt.Errorf("health check has no target (url, port, or command)")
		}

		if checkErr == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}

	return fmt.Errorf("health check timed out after %s", timeout)
}

// WaitForService waits until a service's health check passes.
func WaitForService(ctx context.Context, svc config.Service) error {
	if svc.HealthCheck == nil {
		return nil
	}

	hc := svc.HealthCheck
	timeout, err := config.ParseDuration(hc.Timeout)
	if err != nil {
		timeout = 30 * time.Second
	}
	interval, err := config.ParseDuration(hc.Interval)
	if err != nil {
		interval = 2 * time.Second
	}

	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		var checkErr error

		switch hc.Type {
		case "http":
			checkErr = checkHTTP(ctx, hc.Target)
		case "tcp":
			if svc.Port > 0 && hc.Target == "" {
				checkErr = checkTCP(ctx, svc.Port)
			} else {
				checkErr = checkTCPAddr(ctx, hc.Target)
			}
		case "cmd":
			checkErr = checkCommand(ctx, hc.Target)
		default:
			return fmt.Errorf("unknown health check type: %s (expected: http, tcp, cmd)", hc.Type)
		}

		if checkErr == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}

	return fmt.Errorf(
		"service %q health check timed out after %s\n\n  💡 Check that the service is starting correctly.\n     Logs: .oneconfig/logs/%s.log",
		svc.Name, timeout, svc.Name,
	)
}

// checkHTTP performs an HTTP GET and expects a 2xx response.
func checkHTTP(ctx context.Context, url string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	return nil
}

// checkTCP attempts to dial a TCP port on localhost.
func checkTCP(ctx context.Context, port int) error {
	return checkTCPAddr(ctx, fmt.Sprintf("localhost:%d", port))
}

// checkTCPAddr attempts to dial a TCP address.
func checkTCPAddr(ctx context.Context, addr string) error {
	var d net.Dialer
	d.Timeout = 2 * time.Second
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

// checkCommand runs a shell command and expects exit code 0.
func checkCommand(ctx context.Context, command string) error {
	return shell.CommandContext(ctx, command).Run()
}
