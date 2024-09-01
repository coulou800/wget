package net

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
	"wget/flag"
	"wget/utils"

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
	// fmt.Println(allowedBytes, len(p))
	if int64(len(p)) > allowedBytes && allowedBytes > 0 {
		p = p[:allowedBytes]
	}

	// Read the allowed bytes
	n, err = r.reader.Read(p)
	r.bytesRead += int64(n)
	if allowedBytes > 0 {
		// If we've reached the rate limit, sleep for the remaining time in the time window
		if r.bytesRead >= r.rateLimit {
			time.Sleep(r.timeWindow - elapsed)
		}
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

func GetWithSpeedLimit(p *mpb.Progress, url string, speedLimit int64) ([]byte, error) {
	var contentWriter bytes.Buffer
	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("error while processing the request %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	filename := utils.GetFilenameFromResponse(resp)
	output_path := flag.GetFlagValue(flag.OUTPUT_FLAG).(*string)

	contentLength := resp.ContentLength

	convertedLenght := contentLength / 1024 //kb

	bar := p.AddBar(contentLength,
		mpb.AppendDecorators(
			decor.AverageSpeed(decor.UnitKB, "% .1f"),
			decor.Percentage(decor.WCSyncSpace),
			decor.OnComplete(
				decor.AverageETA(decor.ET_STYLE_GO, decor.WCSyncSpace),
				"✅",
			),
		),
		mpb.BarStyle(" ▓▓░ "),
		mpb.BarNewLineExtend(func(w io.Writer, s *decor.Statistics) {
			if !s.Completed {
				w.Write([]byte(fmt.Sprintf("Downloading: %s size: %v KB | %v", filename, convertedLenght, resp.Status)))

			} else {
				w.Write([]byte(fmt.Sprintf("%s saved into %s", filename, *output_path)))
			}
			w.Write([]byte("\n\n"))
		}),
	)

	reader := bar.ProxyReader(resp.Body)
	limitedReader := NewRateLimitedReader(reader, speedLimit)

	_, err = io.Copy(&contentWriter, limitedReader)

	if err != nil {
		fmt.Printf("error from reading progressReader %v", err)

		return nil, err
	}
	p.Wait()

	body, _ := io.ReadAll(&contentWriter)

	return body, nil
}
