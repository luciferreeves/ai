package commands

import "github.com/bwmarrin/discordgo"

var (
	Commands = []*discordgo.ApplicationCommand{
		{
			Name:        "play",
			Description: "Search and play a song from Spotify or YouTube",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "query",
					Description:  "Search query for the song/playlist (or URL)",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
	}
)
