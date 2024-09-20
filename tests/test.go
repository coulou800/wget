package tests

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"
	"wget/net"
	"wget/state"
	"wget/utils"

	"golang.org/x/time/rate"
)

func TestGetFilenameFromResponse(t *testing.T) {
	resp := &http.Response{
		Request: &http.Request{
			URL: &url.URL{Path: "/path/to/file.txt"},
		},
		Header: http.Header{
			"Content-Disposition": []string{"attachment; filename=\"testfile.txt\""},
		},
	}

	filename := utils.GetFilenameFromResponse(resp)
	if filename != "testfile.txt" {
		t.Errorf("Expected filename 'testfile.txt', got '%s'", filename)
	}
}

func TestConvertedLengthStr(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{1024, "1.0 Kb"},
		{1048576, "1.0 Mb"},
		{1073741824, "1.0 Gb"},
		{500, "500 B"},
	}

	for _, test := range tests {
		result := utils.ConvertedLenghtStr(test.input)
		if result != test.expected {
			t.Errorf("For input %d, expected %s, got %s", test.input, test.expected, result)
		}
	}
}

func TestRateLimitedReader(t *testing.T) {
	// Create a simple reader
	data := []byte("Hello, World!")
	reader := bytes.NewReader(data)

	// Create a rate-limited reader with a limit of 5 bytes per second
	rlReader := net.NewRateLimitedReader(reader, 5)

	buf := make([]byte, 10)
	n, err := rlReader.Read(buf)

	if err != nil && err != io.EOF {
		t.Fatalf("Expected no error, got %v", err)
	}

	if n != 10 {
		t.Errorf("Expected to read 10 bytes, got %d", n)
	}
}

func TestAddLink(t *testing.T) {
	state.InitNewState()
	link := "http://example.com"

	state.AddLink(link)

	// Check if the link was added
	if len(state.GetStates().Mirror.Links) == 0 {
		t.Error("Expected link to be added, but it was not")
	}
}

func TestSetBaseUrl(t *testing.T) {
	state.InitNewState()
	testURL, _ := url.Parse("http://example.com")

	state.SetBaseUrl(testURL)

	if state.GetBaseUrl().String() != testURL.String() {
		t.Errorf("Expected base URL to be %s, got %s", testURL.String(), state.GetBaseUrl().String())
	}
}

func TestLimiter(t *testing.T) {
	limiter := rate.NewLimiter(rate.Every(time.Millisecond*250), 1)

	// Test if the limiter allows a request
	err := limiter.Wait(context.Background())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
