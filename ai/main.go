package main

import (
	"ai/config"
	"ai/types"
	"ai/utils/logger"
	"fmt"
)

func main() {
	logger.Log(fmt.Sprintf("Bot Started. Config: %+v", config.Config), types.LogOptions{})
}
