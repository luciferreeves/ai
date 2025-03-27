package types

type LogLevel string

const (
	Debug   LogLevel = "debug"
	Info    LogLevel = "info"
	Warn    LogLevel = "warn"
	Error   LogLevel = "error"
	Success LogLevel = "success"

	Reset = "\033[0m"
	Cyan  = "\033[36m"
	Gray  = "\033[90m"

	LevelColorInfo    = "\033[34mINFO   \033[0m"
	LevelColorWarn    = "\033[33mWARN   \033[0m"
	LevelColorError   = "\033[31mERROR  \033[0m"
	LevelColorDebug   = "\033[35mDEBUG  \033[0m"
	LevelColorSuccess = "\033[32mSUCCESS\033[0m"

	MessageColorInfo    = "\033[97m"
	MessageColorWarn    = "\033[33m"
	MessageColorError   = "\033[31m"
	MessageColorDebug   = "\033[90m"
	MessageColorSuccess = "\033[32m"
)

type LogOptions struct {
	Timestamp bool
	Prefix    string
	Level     LogLevel
	Fatal     bool
}
