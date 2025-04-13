package commands

import (
	"ai/types"
	"ai/utils/logger"
	"ai/utils/music"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func Play(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	input := options[0].StringValue()

	if input == "min_chars" {
		respondWithError(s, i, "Enter at least 3 characters to search.")
		return
	}

	if input == "no_results" || input == "search_error" {
		respondWithError(s, i, "No results found for your query. Try a different search term.")
		return
	}

	guildID := i.GuildID
	userID := i.Member.User.ID

	isSameVC, userChannelID := music.IsUserInSameVC(s, guildID, userID)

	if userChannelID == "" {
		respondWithError(s, i, "You must be in a voice channel to use this command.")
		return
	}

	voice, exists := music.VoiceConnection[guildID]
	if exists && !isSameVC {
		channel, err := s.Channel(voice.ChannelID)
		if err == nil {
			respondWithError(s, i, fmt.Sprintf("I'm already in the voice channel **%s**. You must be in the same voice channel to control playback.", channel.Name))
		} else {
			respondWithError(s, i, "I'm already in a different voice channel. You must be in the same voice channel to control playback.")
		}
		return
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})

	var trackURL, trackID, trackTitle string
	var sourceType types.SourceType

	if strings.Contains(input, "|") {
		parts := strings.Split(input, "|")
		if len(parts) >= 3 {
			sourceType = types.SourceType(parts[0])
			trackID = parts[1]
			trackURL = parts[2]

			trackInfo, err := music.GetTrackInfo(trackID, sourceType)
			if err != nil {
				trackTitle = "Selected track"
			} else {
				trackTitle = trackInfo.Title
			}

			if sourceType == types.Spotify {
				ytTrack, err := music.GetYouTubeForSpotify(trackInfo.Title, trackInfo.Artist)
				if err != nil {
					updateResponse(s, i, "❌ Error fetching YouTube equivalent for Spotify track.")
					return
				}
				trackURL = ytTrack.URL
				trackID = ytTrack.ID
			}
		} else {
			updateResponse(s, i, "❌ Invalid track selection. Please try again.")
			return
		}
	} else {
		if music.IsYouTubeURL(input) {
			trackInfo, err := music.GetYouTubeInfo(input)
			if err != nil {
				updateResponse(s, i, "❌ Failed to get information for this YouTube URL.")
				return
			}
			trackURL = input
			trackID = trackInfo.ID
			trackTitle = trackInfo.Title
			sourceType = types.YouTube
		} else if music.IsSpotifyURL(input) {
			trackInfo, err := music.GetSpotifyInfo(input)
			if err != nil {
				updateResponse(s, i, "❌ Failed to get information for this Spotify URL.")
				return
			}

			ytTrack, err := music.GetYouTubeForSpotify(trackInfo.Title, trackInfo.Artist)
			if err != nil {
				updateResponse(s, i, "❌ Error fetching YouTube equivalent for Spotify track.")
				return
			}
			trackURL = ytTrack.URL
			trackID = ytTrack.ID
			trackTitle = trackInfo.Title
			sourceType = types.Spotify
		} else {
			results, err := music.Search(input, 1)
			if err != nil || len(results) == 0 {
				updateResponse(s, i, "❌ No results found for your search query.")
				return
			}

			result := results[0]
			trackTitle = result.Title
			trackID = result.ID
			sourceType = result.SourceType

			if result.SourceType == types.Spotify {
				ytTrack, err := music.GetYouTubeForSpotify(result.Title, result.Artist)
				if err != nil {
					updateResponse(s, i, "❌ Error fetching YouTube equivalent for Spotify track.")
					return
				}
				trackURL = ytTrack.URL
				trackID = ytTrack.ID
			} else {
				trackURL = result.URL
			}
		}
	}

	voice, err := music.JoinVoiceChannel(s, guildID, userChannelID)
	if err != nil {
		logger.Log(fmt.Sprintf("Failed to join voice channel: %v", err), types.LogOptions{
			Prefix: "Play Command",
			Level:  types.Error,
		})
		updateResponse(s, i, "❌ Failed to join your voice channel.")
		return
	}

	updateResponse(s, i, fmt.Sprintf("🎵 Now playing: **%s**", trackTitle))

	go func() {
		err := voice.PlayYouTube(trackURL, trackID)
		if err != nil {
			logger.Log(fmt.Sprintf("Failed to play track: %v", err), types.LogOptions{
				Prefix: "Play Command",
				Level:  types.Error,
			})
			updateResponse(s, i, fmt.Sprintf("❌ Error playing **%s**: %v", trackTitle, err))
		}
	}()
}
