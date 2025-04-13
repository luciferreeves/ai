package commands

import (
	"ai/utils/music"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func Disconnect(s *discordgo.Session, i *discordgo.InteractionCreate) {
	guildID := i.GuildID
	userID := i.Member.User.ID

	isSameVC, userChannelID := music.IsUserInSameVC(s, guildID, userID)

	if userChannelID == "" {
		respondWithError(s, i, "You must be in a voice channel to use this command.")
		return
	}

	voice, exists := music.VoiceConnection[guildID]
	if !exists {
		respondWithError(s, i, "I'm not in a voice channel.")
		return
	}

	if !isSameVC {
		channel, err := s.Channel(voice.ChannelID)
		if err == nil {
			respondWithError(s, i, fmt.Sprintf("You must be in the same voice channel as me (**%s**) to use this command.", channel.Name))
		} else {
			respondWithError(s, i, "You must be in the same voice channel as me to use this command.")
		}
		return
	}

	channel, err := s.Channel(voice.ChannelID)
	channelName := "voice channel"
	if err == nil {
		channelName = channel.Name
	}

	err = music.LeaveVoiceChannel(guildID)
	if err != nil {
		respondWithError(s, i, fmt.Sprintf("Error disconnecting from voice channel: %v", err))
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("✅ Disconnected from **%s**.", channelName),
		},
	})
}
