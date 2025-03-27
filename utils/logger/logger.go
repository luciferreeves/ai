package logger

import (
	"fmt"
	"os"
	"strings"
	"time"

	"ai/types"
)

func getTimestamp() string {
	return time.Now().Format(time.RFC3339)
}

func getLevelColor(level types.LogLevel) string {
	switch level {
	case types.Info:
		return types.LevelColorInfo
	case types.Warn:
		return types.LevelColorWarn
	case types.Error:
		return types.LevelColorError
	case types.Debug:
		return types.LevelColorDebug
	case types.Success:
		return types.LevelColorSuccess
	default:
		return types.LevelColorInfo
	}
}

func getMessageColor(level types.LogLevel) string {
	switch level {
	case types.Info:
		return types.MessageColorInfo
	case types.Warn:
		return types.MessageColorWarn
	case types.Error:
		return types.MessageColorError
	case types.Debug:
		return types.MessageColorDebug
	case types.Success:
		return types.MessageColorSuccess
	default:
		return types.MessageColorInfo
	}
}

func Log(message interface{}, options types.LogOptions) {
	var builder strings.Builder

	if options.Level == "" {
		options.Level = types.Info
	}

	if options.Timestamp {
		builder.WriteString(types.Gray)
		builder.WriteString(getTimestamp())
		builder.WriteString(types.Reset)
		builder.WriteString(" ")
	}

	builder.WriteString(getLevelColor(options.Level))
	builder.WriteString(" ")

	if options.Prefix != "" {
		builder.WriteString(types.Cyan)
		builder.WriteString("[")
		builder.WriteString(options.Prefix)
		builder.WriteString("]")
		builder.WriteString(types.Reset)
		builder.WriteString(" ")
	}

	builder.WriteString(getMessageColor(options.Level))

	switch msg := message.(type) {
	case error:
		builder.WriteString(msg.Error())
	case string:
		builder.WriteString(msg)
	default:
		builder.WriteString(fmt.Sprintf("%v", msg))
	}

	builder.WriteString(types.Reset)
	builder.WriteString("\n")

	if options.Level == types.Error || options.Level == types.Warn {
		os.Stderr.WriteString(builder.String())
	} else {
		os.Stdout.WriteString(builder.String())
	}

	if options.Fatal {
		os.Exit(1)
	}
}
