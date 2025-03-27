package main

import (
	"ai/config"
	"ai/types"
	"ai/utils/logger"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

const (
	ProcessPrefix = "Main Process"
)

var (
	session *discordgo.Session
	err     error
)

func init() {
	session, err = discordgo.New("Bot " + config.Config.DiscordToken)
	if err != nil {
		logger.Log("error creating Discord session,", types.LogOptions{Fatal: true, Prefix: ProcessPrefix, Level: types.Error})
	}

	session.Identify.Intents |= discordgo.IntentsAllWithoutPrivileged
	session.AddHandler(ready)
}

func main() {
	err = session.Open()
	if err != nil {
		logger.Log("error opening connection,", types.LogOptions{Fatal: true, Prefix: ProcessPrefix, Level: types.Error})
	}

	// Wait here until CTRL-C or other term signal is received.
	logger.Log("Bot is now running. Press CTRL-C to exit.", types.LogOptions{Prefix: ProcessPrefix})
	defer session.Close()

	session_close := make(chan os.Signal, 1)
	signal.Notify(session_close, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	<-session_close

	logger.Log("Recived SIGINT. Shutting down gracefully.", types.LogOptions{Prefix: ProcessPrefix})
}

func ready(s *discordgo.Session, event *discordgo.Ready) {
	logger.Log("Bot is ready.", types.LogOptions{Prefix: ProcessPrefix, Level: types.Success})

	switch config.Config.Activity {
	case types.PLAYING:
		err = s.UpdateGameStatus(0, config.Config.ActivityMessage)
	case types.WATCHING:
		err = s.UpdateWatchStatus(0, config.Config.ActivityMessage)
	case types.LISTENING:
		err = s.UpdateListeningStatus(config.Config.ActivityMessage)
	case types.STREAMING:
		err = s.UpdateStreamingStatus(0, config.Config.ActivityMessage, config.Config.ActivityURL)
	}

	if err != nil {
		logger.Log("Error attempting to set activity.", types.LogOptions{Prefix: ProcessPrefix, Level: types.Error})
	}
}
