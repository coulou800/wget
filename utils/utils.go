package utils

import (
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/sys/unix"
)

func GetFilenameFromResponse(resp *http.Response) string {

	splitted_url := strings.Split(resp.Request.URL.Path, "/")
	defaultFilename := splitted_url[len(splitted_url)-1]

	cd := resp.Header.Get("Content-Disposition")
	if cd == "" {
		return defaultFilename
	}

	_, params, err := mime.ParseMediaType(cd)
	if err != nil {
		fmt.Println("Error parsing Content-Disposition:", err)
		return defaultFilename
	}

	filename, ok := params["filename"]
	if !ok {
		return defaultFilename
	}

	return strings.TrimSpace(filename)
}

func PathHasDir(dir string, path string) bool {
	re, err := regexp.Compile(dir + "/")
	if err != nil {
		return false
	}

	return re.MatchString(path)
}

func GetCurrentTime() string {
	// Get the current time
	now := time.Now()

	// Format the time
	formattedTime := now.Format("2006-01-02 15:04:05")

	return formattedTime
}

func GetContentLength(url string) int64 {
	client := &http.Client{}
	userAgent := "Mozilla/5.0 (X11; Linux x86_64; rv:128.0) Gecko/20100101 Firefox/128.0"
	req, _ := http.NewRequest("HEAD", url, nil)
	req.Header.Add("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()

	contentLength := resp.ContentLength

	cookies := resp.Cookies()
	for _, v := range cookies {
		fmt.Println(v)
	}

	if contentLength == -1 {
		return 0
	}

	return contentLength
}

func GetTerminalWidth() int {
	fd := int(os.Stdout.Fd())

	ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	if err != nil {
		return 80
	}

	return int(ws.Col)
}
func ByteToKb(val int64) float64 {
	return float64(val) / 1024
}

func ByteToMb(val int64) float64 {
	return float64(val) / math.Pow(1024, 2)
}

func ByteToGb(val int64) float64 {
	return float64(val) / math.Pow(1024, 3)
}

func ConvertedLenghtStr(val int64) string {
	if val >= int64(math.Pow(1024, 3)*0.85) {
		return fmt.Sprintf("%.1f Gb", ByteToGb(val))
	} else if val >= int64(math.Pow(1024, 2)*0.85) {
		return fmt.Sprintf("%.1f Mb", ByteToMb(val))
	} else if float64(val) > float64(1024)*0.85 {
		return fmt.Sprintf("%.1f Kb", ByteToKb(val))
	} else {
		return fmt.Sprintf("%d B", val)
	}
}

func ConvertedRateLimit(valStr string) int64 {
	errMsg := "invalid rate limit. usage: --rate-limit 200k"
	if len(valStr) < 2 {
		fmt.Println(errMsg)
		os.Exit(1)
	}
	valStr = strings.ToLower(valStr)
	m := valStr[len(valStr)-1]
	val, err := strconv.ParseFloat(valStr[:len(valStr)-1], 64)
	if err != nil {
		fmt.Printf("can't convert %v. Please provide valid rate limit\n", valStr[:len(valStr)-1])
		os.Exit(1)
	}

	switch m {
	case 'k':
		return int64(val * 1000)
	case 'm':
		return int64(val * math.Pow(1000, 2))
	case 'g':
		return int64(val * math.Pow(1000, 3))
	default:
		fmt.Println(errMsg)
		os.Exit(1)
	}
	return -1
}

func ExtractURLs(f *os.File) []string {
	content, err := io.ReadAll(f)
	if err != nil {
		fmt.Println("cannot read", err)
	}
	re := regexp.MustCompile(`url\(['"]?(.*?)['"]?\)`)

	matches := re.FindAllStringSubmatch(string(content), -1)

	var urls []string

	for _, match := range matches {
		if len(match) > 1 {
			urls = append(urls, match[1])
		}
	}

	return urls
}
