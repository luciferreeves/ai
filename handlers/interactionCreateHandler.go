package handlers

import "github.com/bwmarrin/discordgo"

func InteractionCreateHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	switch i.Type {
	case discordgo.InteractionApplicationCommand:
		if handler, ok := SlashCommandHandlers[i.ApplicationCommandData().Name]; ok {
			handler(s, i)
		}
	}
}
