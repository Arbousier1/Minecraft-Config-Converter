package main

import (
	"errors"
	"net/http"
	"time"
)

func waitForServerReady(url string, serveErrCh <-chan error, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 500 * time.Millisecond}

	for time.Now().Before(deadline) {
		select {
		case err := <-serveErrCh:
			if err != nil {
				return err
			}
			return nil
		default:
		}

		resp, err := client.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				return nil
			}
		}

		time.Sleep(100 * time.Millisecond)
	}
	return errors.New("server did not become ready in time")
}
