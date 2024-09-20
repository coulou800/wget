package flag

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"wget/utils"
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
	CONVERT_FLAG
	MIRROR_FLAG
	REJECT_FLAG
	URLS_FLAG
	EXCLUDE_FLAG
)

var (
	Output      = new(string)
	Path        = new(string)
	RateLimit   = new(string)
	Input       = new(string)
	Background  = new(bool)
	Content     = new(string)
	RejectedStr = new(string)
	Convert     = new(bool)
	urls        = new([]string)
	rateLimit   int64
	Mirror      = new(bool)
	Reject      = new([]string)
	Excludes    = new([]string)
	flagNames   = make(map[Flag]string)
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
	flagNames[MIRROR_FLAG] = "mirror"
	flagNames[REJECT_FLAG] = "reject"
	flagNames[EXCLUDE_FLAG] = "exclude"
	flagNames[CONVERT_FLAG] = "convert-links"

}

func InitFlagValues() {
	flagsValues[RATELIMIT_FLAG] = RateLimit
	flagsValues[INPUT_FLAG] = Input
	flagsValues[BACKGROUND_FLAG] = Background
	flagsValues[CONTENT_FLAG] = Content
	flagsValues[REJECT_FLAG] = Reject
	flagsValues[EXCLUDE_FLAG] = Excludes
	flagsValues[CONVERT_FLAG] = Convert

	limited := *RateLimit != ""

	if limited {
		rateLimit = utils.ConvertedRateLimit(*RateLimit)
	}
	pwd, _ := os.Getwd()

	if *Path == "" {
		path, err := os.Getwd()
		if err != nil {
			os.Stderr.WriteString(err.Error() + "\n")
			os.Exit(1)
		}
		Path = &path
	} else {
		dir, err := os.Stat(*Path)
		if err != nil {
			dir, err = os.Stat(fmt.Sprintf("%v/%v", pwd, *Path))
			if err != nil {
				err := os.MkdirAll(*Path, 0777)
				if err != nil {
					os.Stderr.WriteString(err.Error() + "\n")
					os.Exit(1)
				}
				dir, err = os.Stat(fmt.Sprintf("%v/%v", pwd, *Path))
				if err != nil {
					println("here")
					os.Stderr.WriteString(err.Error() + "\n")
					os.Exit(1)
				}
			}

		}
		if !dir.IsDir() {
			os.Stderr.WriteString("output dir is not a directory\n")
			os.Exit(1)
		}
	}

	if (*Path)[0] == '~' {
		homeDir, err := os.UserHomeDir()
		p := (*Path)[1:]

		if err == nil {
			absolutePath := filepath.Join(homeDir, p)
			Path = &absolutePath
		}
	}

	if (*Path)[0] != '/' {
		absolutePath := filepath.Join(pwd, *Path)
		Path = &absolutePath
	}

	err := os.MkdirAll(*Path, 0777)
	if err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		os.Exit(1)
	}

	flagsValues[LIMITED_FLAG] = limited
	flagsValues[OUTPUT_FLAG] = Output
	flagsValues[PATH_FLAG] = Path
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
			if line == "" {
				continue
			}
			parsedURL, err := url.Parse(line)
			if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
				os.Stderr.Write([]byte(fmt.Sprintf("Invalid url: %s\n", line)))
				continue
			}
			u = append(u, line)
		}
	} else {
		u = append(u, args[0])
	}

	if len(u) == 0 {
		os.Stderr.WriteString("please provide valid url\n")
		os.Exit(1)
	}
	urls = &u

}

func GetRateLimit() int64 {
	return rateLimit
}

func GetUrls() []string {
	return *urls
}

func GetArgs() []string {
	args := os.Args[1:]
	var a []string
	for _, arg := range args {
		if arg != "-B" && arg != "--background" {
			a = append(a, arg)
		}
	}
	return a
}

func IsMirror() bool {
	return *Mirror
}

func SetOutputPath(path string) {
	flagsValues[PATH_FLAG] = &path
}

func CheckFlags() error {
	if Provided(REJECT_FLAG) || Provided(EXCLUDE_FLAG) || Provided(CONVERT_FLAG) && !*Mirror {
		return fmt.Errorf("should specify mirror flag: --mirror")
	}

	if *Mirror && Provided(INPUT_FLAG) {
		return fmt.Errorf("mirror and input cannot go alongside")
	}

	return nil
}
