package net

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
)

type rateLimitedReader struct {
	reader     io.Reader
	rateLimit  int64
	lastTime   time.Time
	bytesRead  int64
	timeWindow time.Duration
}

func (r *rateLimitedReader) Read(p []byte) (n int, err error) {
	now := time.Now()
	elapsed := now.Sub(r.lastTime)

	// If the elapsed time is greater than or equal to the time window,
	// reset the bytesRead and lastTime.
	if elapsed >= r.timeWindow {
		r.bytesRead = 0
		r.lastTime = now
		elapsed = 0
	}

	// Calculate the allowed bytes to read in this call
	allowedBytes := r.rateLimit - r.bytesRead
	if int64(len(p)) > allowedBytes {
		p = p[:allowedBytes]
	}

	// Read the allowed bytes
	n, err = r.reader.Read(p)
	r.bytesRead += int64(n)

	// If we've reached the rate limit, sleep for the remaining time in the time window
	if r.bytesRead >= r.rateLimit {
		time.Sleep(r.timeWindow - elapsed)
	}

	return n, err
}

func NewRateLimitedReader(r io.Reader, limit int64) io.Reader {
	return &rateLimitedReader{
		reader:     r,
		rateLimit:  limit,
		lastTime:   time.Now(),
		bytesRead:  0,
		timeWindow: time.Second,
	}
}

func GetWithSpeedLimit(url string, speedLimit int64) ([]byte, error) {

	var contentWriter bytes.Buffer

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("error while processing the request %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	p := mpb.New()

	filename := resp.Request.URL.Path

	contentLength := resp.ContentLength

	bar := p.AddBar(contentLength,
		mpb.PrependDecorators(decor.Name(filename, decor.WCSyncSpace)),
		 mpb.AppendDecorators(
		decor.AverageSpeed(decor.UnitKB, "% .1f"),
		decor.Percentage(decor.WCSyncSpace),
	),
		// mpb.BarNewLineExtend(func(w io.Writer, s *decor.Statistics) {
	
		// })
		mpb.BarStyle("▓▓▓░░"),
		// mpb.BarRemoveOnComplete(),
	)
	fmt.Println(contentLength)
	// Create a progress reader to track the download progress
	reader := bar.ProxyReader(resp.Body)
	limitedReader := NewRateLimitedReader(reader, speedLimit)

	_, err = io.Copy(&contentWriter, limitedReader)

	if err != nil {
		fmt.Printf("error from reading progressReader %v", err)
		return nil, err
	}
	p.Wait()
	fmt.Println("Download complete")
	body, _ := io.ReadAll(&contentWriter)
	return body, nil
}

// ProgressReader is a custom io.Reader that tracks the progress of reading
type ProgressReader struct {
	Reader        io.ReadWriter
	bytes         int64
	rate          int64
	RateLimit     int64
	lastReadTime  time.Time
	contentLength int64
}

// Read reads from the underlying reader and updates the progress
func (pr *ProgressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.rate = int64(n)
	time.Sleep(time.Duration(float64(n) / float64(pr.RateLimit) * float64(time.Second)))
	return n, err
}
