package config

import (
	"ai/types"
	"ai/utils/logger"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

var Config *types.BotConfig

func init() {
	logPrefix := "Config"
	logOptions := types.LogOptions{
		Prefix: logPrefix,
		Level:  types.Error,
		Fatal:  true,
	}

	if err := godotenv.Load(); err != nil {
		logger.Log("Failed to load environment variables", logOptions)
	}

	Config = &types.BotConfig{
		GuildID:             getEnv("GUILD_ID"),
		DiscordToken:        getEnv("DISCORD_TOKEN"),
		SpotifyClientId:     getEnv("SPOTIFY_CLIENT_ID"),
		SpotifyClientSecret: getEnv("SPOTIFY_CLIENT_SECRET"),
		YoutubeAPIKey:       getEnv("YOUTUBE_API_KEY"),
		Activity:            types.ActivityType(getIntEnv("ACTIVITY")),
		ActivityMessage:     getEnv("ACTIVITY_MESSAGE"),
		ActivityURL:         getEnv("ACTIVITY_URL"),
	}

	if Config.GuildID == "" {
		logger.Log("Unable to read Guild ID. environment variable GUILD_ID is required", logOptions)
	}

	if Config.DiscordToken == "" {
		logger.Log("Unable to read Discord token. environment variable DISCORD_TOKEN is required", logOptions)
	}

	if Config.SpotifyClientId == "" {
		logger.Log("Unable to read Spotify client ID. environment variable SPOTIFY_CLIENT_ID is required", logOptions)
	}

	if Config.SpotifyClientSecret == "" {
		logger.Log("Unable to read Spotify client secret. environment variable SPOTIFY_CLIENT_SECRET is required", logOptions)
	}

	if Config.YoutubeAPIKey == "" {
		logger.Log("Unable to read YouTube API key. environment variable YOUTUBE_API_KEY is required", logOptions)
	}

	logOptions.Level = types.Warn
	logOptions.Fatal = false
	if Config.Activity == 0 {
		logger.Log("Activity message is empty or not set. Defaulting to PLAYING", logOptions)
		Config.Activity = types.PLAYING
	}

	if Config.ActivityMessage == "" {
		logger.Log("Activity message is empty or not set. Defaulting to empty string", logOptions)
		Config.ActivityMessage = ""
	}

	if Config.Activity == types.STREAMING && Config.ActivityURL == "" {
		logger.Log("Activity URL is empty or not set. Defaulting to empty string", logOptions)
		Config.ActivityURL = ""
	}

	logOptions.Level = types.Success
	logOptions.Fatal = false
	logger.Log("Config loaded successfully", logOptions)
}

func getEnv(key string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return ""
	}
	return strings.TrimSpace(value)
}

func getIntEnv(key string) int {
	value := getEnv(key)
	if value == "" {
		return 0
	}
	i, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return i
}
