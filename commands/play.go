package commands

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// Command: Play
// Takes in a search query and autocompletes the search query to play a song from Spotify or YouTube
// Also takes in a URL to a Spotify Song/Playlist or YouTube Video/Playlist
// Joins the user's voice channel and plays the song. Adds to the queue if a song is already playing
// in any of the voice channels. User must be in a voice channel to use this command
// Usage: /play <search query or URL>
func Play(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// For now return the query as reply
	reply := fmt.Sprintf("Playing: %s", i.ApplicationCommandData().Options[0].StringValue())
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: reply,
		},
	})
}
