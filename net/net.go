package net

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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

func GetWithSpeedLimit(p *mpb.Progress, u string, speedLimit int64) {
	client := &http.Client{}
	fileInfos := GetFileInfos(u)
	contentLength := fileInfos.ContentLenght
	userAgent := "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0"
	req, _ := http.NewRequest("GET", u, nil)
	req.Header.Add("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var added bool

	if resp.StatusCode != 200 {
		errMsg := fmt.Errorf("couldn't get %s. reason: %v", u, resp.Status)
		fmt.Printf("%v\n\n", errMsg)
		return
	}

	var filename string
	if flag.Provided(flag.OUTPUT_FLAG) {
		filename = *flag.GetFlagValue(flag.OUTPUT_FLAG).(*string)
	} else {
		filename = fileInfos.FileName
	}

	output_path := flag.GetFlagValue(flag.PATH_FLAG).(*string)
	var path string

	parsedURL, _ := url.Parse(u)
	if flag.IsMirror() {
		filePath := filepath.Join(*output_path, filepath.Base(parsedURL.Path))
		if filePath == *output_path {
			filename = "index.html"
			filePath = filepath.Join(*output_path, filename)
		} else {
			filePath = filepath.Join(*output_path, parsedURL.Path)
		}
		path = filePath
	} else {
		path = filepath.Join(*output_path, filename)
	}

	if strings.Contains(fileInfos.ContentType, "text/html") && filepath.Ext(path) != ".html" {
		path += ".html"
	}

	out_file, err := os.Create(path)
	if err != nil {
		return
	}
	defer out_file.Close()

	// Common processing logic (for both foreground and background modes)
	// This will ensure file processing works even if there's no progress bar (background mode)
	processFile := func() {
		if flag.IsMirror() && !added {
			f := state.FileToProcess{
				Path: path,
				Url:  parsedURL,
			}
			state.AddToReadyExtract(f)
			added = true
		}
	}

	// Check if running in background
	if state.IsBackground() {
		// // Redirect os.Stdout to a specific file
		// logFile, err := os.OpenFile("background_output.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		// if err != nil {
		// 	fmt.Println("Failed to open log file:", err)
		// 	return
		// }
		// defer logFile.Close()
		// os.Stdout = logFile
		// os.Stderr = logFile

		// No progress bar in the background, but still process the file
		limitedReader := NewRateLimitedReader(resp.Body, speedLimit)
		_, err = io.Copy(out_file, limitedReader)
		if err != nil {
			fmt.Println("here", err)
			return
		}

		// Call file processing in the background as well
		processFile()

	} else {
		// Foreground logic with progress bar
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
			return
		}

		// Call file processing in the foreground after the download completes
		processFile()
	}
}

type FileInfos struct {
	ContentType   string
	ContentLenght int64
	FileName      string
}

func GetFileInfos(url string) FileInfos {
	client := &http.Client{}
	userAgent := "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0"
	req, _ := http.NewRequest("HEAD", url, nil)
	req.Header.Add("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return FileInfos{}
	}
	defer resp.Body.Close()

	contentLength := resp.ContentLength
	contentType := resp.Header.Get("Content-Type")

	var filename string
	var defaultFilename string

	splitted_url := strings.Split(resp.Request.URL.Path, "/")
	if strings.HasSuffix(resp.Request.URL.Path, "/") {
		defaultFilename = splitted_url[len(splitted_url)-2]

	} else {
		defaultFilename = splitted_url[len(splitted_url)-1]

	}
	filename = defaultFilename
	cd := resp.Header.Get("Content-Disposition")
	if cd == "" {
		filename = defaultFilename
	}

	_, params, err := mime.ParseMediaType(cd)
	if err != nil {
		filename = defaultFilename
	}

	filename, ok := params["filename"]
	if !ok || filename == "" {
		filename = defaultFilename
	}

	return FileInfos{
		ContentType:   contentType,
		ContentLenght: contentLength,
		FileName:      filename,
	}
}
