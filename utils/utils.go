package utils

import (
	"bufio"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"net/url"
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

func ExtractURLs(baseUrl *url.URL, f io.Reader) []string {
	re := regexp.MustCompile(`url\(['"]?(.*?)['"]?\)`)
	content, _ := io.ReadAll(f)

	matches := re.FindAllStringSubmatch(string(content), -1)

	var urls []string

	for _, match := range matches {
		if len(match) > 1 {
			// p := ResolveRelativePath(baseUrl, match[1])
			// println(p)
			urls = append(urls, match[1])
		}
	}

	return urls
}

// func ResolveRelativePath(baseUrl *url.URL, path string) string {
// 	var url *url.URL
// 	// if IsSameDomain(baseUrl, path) {
// 	url, _ = url.Parse(ResolveLink(baseUrl, path))
// 	// } else {
// 	// 	url, _ = url.Parse(path)
// 	// }

// 	return url.Path
// }

func IsSameDomain(baseUrl *url.URL, link string) bool {
	linkUrl, err := url.Parse(link)
	if err != nil {
		return false
	}
	return baseUrl.Hostname() == linkUrl.Hostname()
}

func ResolveLink(baseUrl *url.URL, link string) string {
	resolvedUrl, err := baseUrl.Parse(link)
	if err != nil {
		return ""
	}
	return resolvedUrl.String()
}

func ReplaceURLsInFile(filename string) error {
	// Open the file for reading
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read file contents line by line
	var fileContent strings.Builder
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fileContent.WriteString(scanner.Text())
		fileContent.WriteString("\n") // Add newline to preserve formatting
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Define a regular expression to match URLs inside "url('')"
	re := regexp.MustCompile(`url\(['"]?(.*?)['"]?\)`)

	// Function to replace URLs with relative paths
	replacer := func(match string) string {
		// Extract the URL from the match
		parts := re.FindStringSubmatch(match)
		if len(parts) > 1 {
			// Replace with relative path by concatenating with basePath
			relativePath := parts[1]
			if strings.HasPrefix(relativePath, "/") {
				relativePath = "." + relativePath
			}
			return fmt.Sprintf("url('%s')", relativePath)
		}
		return match
	}

	// Replace all matches using the replacer function
	modifiedContent := re.ReplaceAllStringFunc(fileContent.String(), replacer)

	// Open the same file for writing and truncate it (clear the content)
	file, err = os.OpenFile(filename, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the modified content back to the file
	_, err = file.WriteString(modifiedContent)
	if err != nil {
		return err
	}

	return nil
}
