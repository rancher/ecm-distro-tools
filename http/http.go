package http

import (
	"net/http"
	"time"
)

func NewClient(timeout time.Duration) http.Client {
	return http.Client{Timeout: timeout}
}
