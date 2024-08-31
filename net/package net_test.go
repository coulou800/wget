package net_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

"wget/net"
)

func TestGetWithSpeedLimit(t *testing.T) {
	// Create a test server to mock the HTTP response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set the response body
		w.Write([]byte("Hello, World!"))
	}))
	defer server.Close()

	// Set the URL of the test server
	url := server.URL

	// Call the function being tested
	body, err := net.GetWithSpeedLimit(url,200)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify the response body
	expectedBody := "Hello, World!"
	if string(body) != expectedBody {
		t.Errorf("unexpected response body, got: %s, want: %s", body, expectedBody)
	}
}

func TestGetWithSpeedLimit_Timeout(t *testing.T) {
	// Create a test server that waits for a long time to simulate a timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Minute)
	}))
	defer server.Close()

	// Set the URL of the test server
	url := server.URL

	// Call the function being tested
	_, err := net.GetWithSpeedLimit(url,200)
	if err == nil {
		t.Error("expected an error, but got nil")
	}
}
