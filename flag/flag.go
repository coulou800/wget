package flag

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type Flag = int

const (
	OUTPUT_FLAG Flag = iota
	PATH_FLAG
	RATELIMIT_FLAG
	INPUT_FLAG
	BACKGROUND_FLAG
	CONTENT_FLAG
	LIMITED_FLAG

	URLS_FLAG
)

var (
	Output     = new(string)
	Path       = new(string)
	RateLimit  = new(string)
	Input      = new(string)
	Background = new(bool)
	Content    = new(string)

	urls      = new([]string)
	rateLimit int64

	flagNames = make(map[Flag]string)
)

var flagsValues = map[int]any{}

func Provided(flagName Flag) bool {
	if v, ok := flagsValues[flagName].(*string); ok {
		return *v != ""
	}
	if v, ok := flagsValues[flagName].(*bool); ok {
		return *v
	}
	return false
}

func GetFlagValue(flag Flag) any {
	return flagsValues[flag]
}

func GetFlagName(flag Flag) string {
	return flagNames[flag]
}

func SetupFlagName() {
	flagNames[OUTPUT_FLAG] = "output"
	flagNames[PATH_FLAG] = "path"
	flagNames[RATELIMIT_FLAG] = "rate-limit"
	flagNames[INPUT_FLAG] = "input"
	flagNames[BACKGROUND_FLAG] = "background"
	flagNames[CONTENT_FLAG] = "content"
	flagNames[LIMITED_FLAG] = "limited"
	flagNames[URLS_FLAG] = "url"
}

func InitFlagValues() {
	flagsValues[PATH_FLAG] = Path
	flagsValues[RATELIMIT_FLAG] = RateLimit
	flagsValues[INPUT_FLAG] = Input
	flagsValues[BACKGROUND_FLAG] = Background
	flagsValues[CONTENT_FLAG] = Content

	limited := *RateLimit != ""

	if limited {
		n, err := strconv.Atoi(*RateLimit)
		if err != nil {
			fmt.Printf("invalid rate limit %v", *RateLimit)
			os.Exit(1)
		}
		rateLimit = int64(n)
	}

	if *Output == "" {
		path, err := os.Getwd()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		Output = &path
	}

	flagsValues[LIMITED_FLAG] = limited
	flagsValues[OUTPUT_FLAG] = Output

}

func SetupUrls(args []string) {
	path := GetFlagValue(INPUT_FLAG).(*string)
	var u []string
	if *path != "" {
		file, err := os.Open(*path)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		scanner := bufio.NewScanner(file)

		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			_, err := url.Parse(line)
			if err != nil {
				os.Stderr.Write([]byte(fmt.Sprintf("Invalid url: %s", line)))
			}
			u = append(u, line)
		}
	} else {
		u = append(u, args[0])
	}
	urls = &u

	fmt.Println(urls)
}

// use this function to get the rate limit in numerical value
func GetRateLimit() int64 {
	return rateLimit
}

func GetUrls() []string {
	return *urls
}
