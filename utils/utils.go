package utils

import (
	"fmt"
	"mime"
	"net/http"
	"strings"
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
