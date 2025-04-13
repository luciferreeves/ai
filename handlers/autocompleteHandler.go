package handlers

import (
	"ai/commands"

	"github.com/bwmarrin/discordgo"
)

var (
	AutocompleteHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"play": commands.PlayAutocomplete,
	}
)
