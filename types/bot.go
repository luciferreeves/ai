package types

type ActivityType int

const (
	PLAYING ActivityType = iota
	LISTENING
	WATCHING
	STREAMING
)

type BotConfig struct {
	GuildID             string
	DiscordToken        string
	SpotifyClientId     string
	SpotifyClientSecret string
	YoutubeAPIKey       string
	Activity            ActivityType
	ActivityMessage     string
	ActivityURL         string
}
