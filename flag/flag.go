package flag


const (
	OUTPUT     = "output"
	PATH   = "path"
	RATELIMIT  = "rate-limit"
	INPUT      = "input"
	BACKGROUND = "background"
	CONTENT    = "content"
)

var (
	Output     string
	Path   string
	RateLimit  string
	Input      string
	Background bool
	Content    string
)

var flags = map[string]any{}

func init()  {
	flags[OUTPUT] = &Output
	flags[PATH] = &Path
	flags[RATELIMIT] = &RateLimit
	flags[INPUT] = &Input
	flags[BACKGROUND] = &Background
	flags[CONTENT] = &Content
}

func Provided(flagName string) bool {
	if v,ok := flags[flagName].(*string);ok {
		return *v != ""
	}
	if v,ok := flags[flagName].(*bool);ok {
		return *v
	}
	return false
}

func GetValue(flagName string) any {
	return flags[flagName]
}


