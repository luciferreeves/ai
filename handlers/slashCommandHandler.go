package handlers

import (
	"ai/commands"

	"github.com/bwmarrin/discordgo"
)

var (
	SlashCommandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"play": commands.Play,
	}
)
