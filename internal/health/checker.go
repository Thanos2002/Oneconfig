// Package health implements health check strategies (HTTP, TCP, shell command)
// with configurable timeout and retry intervals.
package health

import (
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"time"

	"github.com/Thanos2002/Oneconfig/internal/config"
)

// Check performs a single health check as defined in the config.
func Check(hc config.HealthCheck) error {
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
		var checkErr error

		switch {
		case hc.URL != "":
			checkErr = checkHTTP(hc.URL)
		case hc.Port != 0:
			checkErr = checkTCP(hc.Port)
		case hc.Command != "":
			checkErr = checkCommand(hc.Command)
		default:
			return fmt.Errorf("health check has no target (url, port, or command)")
		}

		if checkErr == nil {
			return nil
		}

		time.Sleep(interval)
	}

	return fmt.Errorf("health check timed out after %s", timeout)
}

// WaitForService waits until a service's health check passes.
func WaitForService(svc config.Service) error {
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
		var checkErr error

		switch hc.Type {
		case "http":
			checkErr = checkHTTP(hc.Target)
		case "tcp":
			if svc.Port > 0 && hc.Target == "" {
				checkErr = checkTCP(svc.Port)
			} else {
				checkErr = checkTCPAddr(hc.Target)
			}
		case "cmd":
			checkErr = checkCommand(hc.Target)
		default:
			return fmt.Errorf("unknown health check type: %s (expected: http, tcp, cmd)", hc.Type)
		}

		if checkErr == nil {
			return nil
		}

		time.Sleep(interval)
	}

	return fmt.Errorf(
		"service %q health check timed out after %s\n\n  💡 Check that the service is starting correctly.\n     Logs: .oneconfig/logs/%s.log",
		svc.Name, timeout, svc.Name,
	)
}

// checkHTTP performs an HTTP GET and expects a 2xx response.
func checkHTTP(url string) error {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
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
func checkTCP(port int) error {
	return checkTCPAddr(fmt.Sprintf("localhost:%d", port))
}

// checkTCPAddr attempts to dial a TCP address.
func checkTCPAddr(addr string) error {
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

// checkCommand runs a shell command and expects exit code 0.
func checkCommand(command string) error {
	cmd := exec.Command("sh", "-c", command)
	return cmd.Run()
}
