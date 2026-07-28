package main

import (
	"errors"
	"net/http"
	"time"
)

const readinessURL = "http://127.0.0.1:8080/health/ready"

var errReadinessUnavailable = errors.New("readiness unavailable")

func checkReadiness() error {
	client := &http.Client{
		Timeout: 2 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return probeReadiness(client, readinessURL)
}

func probeReadiness(client *http.Client, target string) error {
	response, err := client.Get(target)
	if err != nil {
		return errReadinessUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return errReadinessUnavailable
	}
	return nil
}
