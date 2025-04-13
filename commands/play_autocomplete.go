package commands

import (
	"ai/types"
	"ai/utils/logger"
	"ai/utils/music"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

func PlayAutocomplete(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var focusedOption *discordgo.ApplicationCommandInteractionDataOption

	for _, option := range i.ApplicationCommandData().Options {
		if option.Focused {
			focusedOption = option
			break
		}
	}

	if focusedOption == nil {
		return
	}

	query := focusedOption.StringValue()

	if len(query) < 3 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Please enter at least 3 characters",
						Value: "min_chars",
					},
				},
			},
		})
		return
	}

	// Search for tracks
	results, err := music.Search(query, 10)
	if err != nil {
		logger.Log(fmt.Sprintf("Search error: %v", err), types.LogOptions{
			Prefix: "Play Autocomplete",
			Level:  types.Error,
		})

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionApplicationCommandAutocompleteResult,
			Data: &discordgo.InteractionResponseData{
				Choices: []*discordgo.ApplicationCommandOptionChoice{
					{
						Name:  "Error searching. Try again later.",
						Value: "search_error",
					},
				},
			},
		})
		return
	}

	// Create choices for autocomplete
	choices := make([]*discordgo.ApplicationCommandOptionChoice, 0, 25)

	for i, result := range results {
		if i >= 25 {
			break // Discord limits to 25 choices
		}

		// Format display name
		var displayName string
		if result.SourceType == types.YouTube {
			displayName = fmt.Sprintf("▶️ %s - %s", result.Title, result.Artist)
		} else {
			displayName = fmt.Sprintf("🎵 %s - %s", result.Title, result.Artist)
		}

		// Truncate name if needed
		if len(displayName) > 100 {
			displayName = displayName[:97] + "..."
		}

		// Use just source type and ID as value to avoid length issues
		valueStr := fmt.Sprintf("%s|%s|%s", result.SourceType, result.ID, result.URL)
		if len(valueStr) > 100 {
			valueStr = fmt.Sprintf("%s|%s", result.SourceType, result.ID)
		}

		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  displayName,
			Value: valueStr,
		})
	}

	if len(choices) == 0 {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{
			Name:  "No results found",
			Value: "no_results",
		})
	}

	// Send response
	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{
			Choices: choices,
		},
	})

	if err != nil {
		logger.Log(fmt.Sprintf("Failed to send autocomplete response: %v", err), types.LogOptions{
			Prefix: "Play Autocomplete",
			Level:  types.Error,
		})
	}
}
