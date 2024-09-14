package state

import (
	"net/url"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type MirrorState struct {
	BaseUrl       *url.URL
	Links         chan string
	FileToProcess chan FileToProcess
	VisitedLinks  *sync.Map
	limiter       *rate.Limiter
}

type FileToProcess struct {
	Path string
	Url  *url.URL
}

type States struct {
	Mirror MirrorState
}

var states States

func GetStates() *States {
	return &states
}

func InitNewState() {
	limiter := rate.NewLimiter(rate.Every(time.Millisecond*250), 1) // Allow 1 request per second

	states = States{
		Mirror: MirrorState{
			VisitedLinks:  &sync.Map{},
			Links:         make(chan string),
			FileToProcess: make(chan FileToProcess),
			limiter:       limiter,
		},
	}
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
