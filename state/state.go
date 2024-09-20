package state

import (
	"net/url"
	"os"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type MirrorState struct {
	BaseUrl        *url.URL
	Links          chan string
	FileToProcess  chan FileToProcess
	VisitedLinks   *sync.Map
	limiter        *rate.Limiter
	ReadyToExtract chan FileToProcess
	URLMap         *sync.Map
}

type FileToProcess struct {
	Path string
	Url  *url.URL
}

type States struct {
	Mirror  MirrorState
	Aborted chan string
}

var states States

func GetStates() *States {
	return &states
}

func InitNewState() {
	limiter := rate.NewLimiter(rate.Every(time.Millisecond*250), 1) // Allow 1 request per second

	states = States{
		Mirror: MirrorState{
			VisitedLinks:   &sync.Map{},
			Links:          make(chan string),
			FileToProcess:  make(chan FileToProcess),
			limiter:        limiter,
			ReadyToExtract: make(chan FileToProcess),
			URLMap:         &sync.Map{},
		},
		Aborted: make(chan string),
	}
}

func MapUrlPath(f FileToProcess) {
	states.Mirror.URLMap.Store(f.Url, f.Path)
}

// func GetPath() string {
// 	return
// }

func IsBackground() bool {
	return os.Getenv("WGET_BACKGROUND") == "1"
}

func AddFileToProcess(f FileToProcess) {
	states.Mirror.FileToProcess <- f
}

func AddLink(link string) {
	states.Mirror.Links <- link
}

func SetBaseUrl(url *url.URL) {
	states.Mirror.BaseUrl = url
}

func GetBaseUrl() *url.URL {
	return states.Mirror.BaseUrl
}

func GetVisitedLinks() *sync.Map {
	return states.Mirror.VisitedLinks
}

func GetLimiter() *rate.Limiter {
	return states.Mirror.limiter
}

func SetVisitedLink(link string) {
	states.Mirror.VisitedLinks.Store(link, true)
}

func GetReadyExtract() chan FileToProcess {
	return states.Mirror.ReadyToExtract
}

func AddToReadyExtract(f FileToProcess) {
	states.Mirror.ReadyToExtract <- f
}

func Abort(u string) {
	states.Aborted <- u
}
