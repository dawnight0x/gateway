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
	Restart         func() error
	Shutdown        func()
}

type Status struct {
	Healthy         bool
	Readiness       string
	ReadinessReason string
	ActiveKeys      int
	FailedKeys      int
	TodayRequests   int
	TodayTokens     int
	Error           string
}

func Supported() bool {
	return false
}

func Run(Options) {}

func Quit() {}
