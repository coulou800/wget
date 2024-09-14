package net

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"
	"wget/flag"
	"wget/state"
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
	completed  bool
}

func (r *rateLimitedReader) Completed() bool {
	return r.completed
}

func (r *rateLimitedReader) Read(p []byte) (n int, err error) {

	now := time.Now()
	elapsed := now.Sub(r.lastTime)

	if elapsed >= r.timeWindow {
		r.bytesRead = 0
		r.lastTime = now
		elapsed = 0
	}

	allowedBytes := r.rateLimit - r.bytesRead
	if int64(len(p)) > allowedBytes && allowedBytes > 0 {
		p = p[:allowedBytes]
	}

	n, err = r.reader.Read(p)
	if err == io.EOF {
		r.completed = true
	}
	r.bytesRead += int64(n)
	if allowedBytes > 0 {
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
		completed:  false,
	}
}

func GetWithSpeedLimit(p *mpb.Progress, u string, speedLimit int64) error {
	client := &http.Client{}
	contentLength := utils.GetContentLength(u)
	userAgent := "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0"
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Add("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		errMsg := fmt.Errorf("couldn't get %s. reason: %v", u, resp.Status)
		fmt.Printf("%v\n\n", errMsg)
		return errMsg
	}

	var filename string

	if flag.Provided(flag.OUTPUT_FLAG) {
		filename = *flag.GetFlagValue(flag.OUTPUT_FLAG).(*string)
	} else {
		filename = utils.GetFilenameFromResponse(resp)
	}

	output_path := flag.GetFlagValue(flag.PATH_FLAG).(*string)
	var path string

	parsedURL, _ := url.Parse(u)
	if flag.IsMirror() {
		// fmt.Println("mirror path", *output_path)
		filePath := filepath.Join(*output_path, filepath.Base(parsedURL.Path))
		// fmt.Println("mirror", filePath, *output_path)

		if filePath == *output_path {
			filename = "index.html"
			filePath = filepath.Join(*output_path, "index.html")

		} else {
			filePath = filepath.Join(*output_path, parsedURL.Path)
		}
		path = filePath

	} else {
		path = filepath.Join(*output_path, filename)
	}

	out_file, err := os.Create(path)
	if err != nil {
		return err
	}
	convertedLenght := utils.ConvertedLenghtStr(contentLength)

	bar := p.AddBar(contentLength,
		mpb.BarWidth(int(float32(utils.GetTerminalWidth())*0.45)),
		mpb.OptionOnCondition(
			mpb.AppendDecorators(
				decor.AverageSpeed(decor.UnitKB, "% .1f"),
				decor.Percentage(decor.WCSyncSpace),
				decor.OnComplete(
					decor.AverageETA(decor.ET_STYLE_GO, decor.WCSyncSpace),
					"✅",
				),
			),

			func() bool {
				return resp.StatusCode == 200
			},
		),
		mpb.OptionOnCondition(
			mpb.BarStyle(" ▓▓░ "),
			func() bool {
				return resp.StatusCode == 200
			},
		),
		mpb.BarNewLineExtend(func(w io.Writer, s *decor.Statistics) {
			if !s.Completed {
				w.Write([]byte(fmt.Sprintf("Downloading: %s | %v / %v | %v", filename, utils.ConvertedLenghtStr(s.Current), convertedLenght, resp.Status)))
			} else {
				w.Write([]byte(fmt.Sprintf("%s saved into %s", filename, *output_path)))
			}
			w.Write([]byte("\n\n"))
		}),
	)

	reader := bar.ProxyReader(resp.Body)
	limitedReader := NewRateLimitedReader(reader, speedLimit)

	_, err = io.Copy(out_file, limitedReader)

	if err != nil {
		fmt.Println("here", err)
		return err
	}
	if flag.IsMirror() {
		f := state.FileToProcess{
			Path: filepath.Join(*output_path, filename),
			Url:  parsedURL,
		}
		state.AddFileToProcess(f)
	}
	return nil
}
