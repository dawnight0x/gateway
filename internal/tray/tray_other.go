//go:build !windows

package tray

type Options struct {
	AdminURL        string
	AdminOpenURL    string
	DataDir         string
	LogPath         string
	Status          func() Status
	OpenAIConfig    string
	AnthropicConfig string
	GeminiConfig    string
	Restart         func()
	Shutdown        func()
}

type Status struct {
	Healthy       bool
	ActiveKeys    int
	FailedKeys    int
	TodayRequests int
	TodayTokens   int
	Error         string
}

func Supported() bool {
	return false
}

func Run(Options) {}

func Quit() {}
