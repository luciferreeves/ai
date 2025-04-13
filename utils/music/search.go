package music

import (
	"ai/config"
	"ai/types"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
)

var (
	youtubeRegex = regexp.MustCompile(`^(https?://)?(www\.)?(youtube\.com|youtu\.?be)/.+`)
	spotifyRegex = regexp.MustCompile(`^(https?://)?(open\.)?spotify\.com/.+`)
)

func IsYouTubeURL(input string) bool {
	return youtubeRegex.MatchString(input)
}

func IsSpotifyURL(input string) bool {
	return spotifyRegex.MatchString(input)
}

func Search(query string, limit int) ([]types.MusicSearchResult, error) {
	var wg sync.WaitGroup
	wg.Add(2)

	var youtubeResults []types.MusicSearchResult
	var spotifyResults []types.MusicSearchResult
	var youtubeErr, spotifyErr error

	go func() {
		defer wg.Done()
		youtubeResults, youtubeErr = SearchYouTube(query, limit/2)
	}()

	go func() {
		defer wg.Done()
		spotifyResults, spotifyErr = SearchSpotify(query, limit/2)
	}()

	wg.Wait()

	if youtubeErr != nil && spotifyErr != nil {
		return nil, fmt.Errorf("both search errors: youtube: %w, spotify: %w", youtubeErr, spotifyErr)
	}

	results := []types.MusicSearchResult{}

	maxLength := max(len(spotifyResults), len(youtubeResults))

	for i := range maxLength {
		if i < len(youtubeResults) {
			results = append(results, youtubeResults[i])
		}
		if i < len(spotifyResults) {
			results = append(results, spotifyResults[i])
		}
	}

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

func SearchSpotify(query string, limit int) ([]types.MusicSearchResult, error) {
	token, err := getSpotifyToken()
	if err != nil {
		return nil, fmt.Errorf("spotify token error: %w", err)
	}

	searchURL := fmt.Sprintf("https://api.spotify.com/v1/search?q=%s&type=track&limit=%d", url.QueryEscape(query), limit)
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("request creation error: %w", err)
	}

	req.Header.Add("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("response read error: %w", err)
	}

	var searchResponse types.SpotifySearchResponse
	if err := json.Unmarshal(body, &searchResponse); err != nil {
		return nil, fmt.Errorf("json unmarshal error: %w", err)
	}

	results := []types.MusicSearchResult{}

	for _, item := range searchResponse.Tracks.Items {
		artistName := ""
		if len(item.Artists) > 0 {
			artistName = item.Artists[0].Name
		}

		thumbnailURL := ""
		if len(item.Album.Images) > 0 {
			thumbnailURL = item.Album.Images[0].URL
		}

		// Format duration as mm:ss
		durationSec := item.DurationMs / 1000
		duration := fmt.Sprintf("%02d:%02d", durationSec/60, durationSec%60)

		results = append(results, types.MusicSearchResult{
			Title:      item.Name,
			Artist:     artistName,
			URL:        item.ExternalUrls.Spotify,
			ID:         item.ID,
			Duration:   duration,
			Thumbnail:  thumbnailURL,
			SourceType: types.Spotify,
		})
	}

	return results, nil
}

func SearchYouTube(query string, limit int) ([]types.MusicSearchResult, error) {
	apiKey := config.Config.YoutubeAPIKey
	searchURL := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/search?part=snippet&q=%s&key=%s&maxResults=%d&type=video",
		url.QueryEscape(query), apiKey, limit,
	)

	resp, err := http.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("search request error: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("response read error: %w", err)
	}

	var searchResponse types.YouTubeSearchResponse
	if err := json.Unmarshal(body, &searchResponse); err != nil {
		return nil, fmt.Errorf("json unmarshal error: %w", err)
	}

	results := []types.MusicSearchResult{}

	for _, item := range searchResponse.Items {
		videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", item.ID.VideoID)

		results = append(results, types.MusicSearchResult{
			Title:      item.Snippet.Title,
			Artist:     item.Snippet.ChannelTitle,
			URL:        videoURL,
			ID:         item.ID.VideoID,
			Duration:   "00:00", // YouTube API requires a separate call to get duration
			Thumbnail:  item.Snippet.Thumbnails.High.URL,
			SourceType: types.YouTube,
		})
	}

	return results, nil
}

func GetTrackInfo(id string, sourceType types.SourceType) (types.MusicSearchResult, error) {
	if sourceType == types.YouTube {
		return GetYouTubeInfoByID(id)
	} else if sourceType == types.Spotify {
		return GetSpotifyInfoByID(id)
	}

	return types.MusicSearchResult{}, fmt.Errorf("unsupported source type: %s", sourceType)
}

func GetYouTubeInfoByID(videoID string) (types.MusicSearchResult, error) {
	apiKey := config.Config.YoutubeAPIKey
	apiURL := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/videos?part=contentDetails,snippet&id=%s&key=%s",
		videoID, apiKey,
	)

	resp, err := http.Get(apiURL)
	if err != nil {
		return types.MusicSearchResult{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.MusicSearchResult{}, err
	}

	var response struct {
		Items []struct {
			Snippet struct {
				Title        string `json:"title"`
				ChannelTitle string `json:"channelTitle"`
				Thumbnails   struct {
					High struct {
						URL string `json:"url"`
					} `json:"high"`
				} `json:"thumbnails"`
			} `json:"snippet"`
			ContentDetails struct {
				Duration string `json:"duration"`
			} `json:"contentDetails"`
		} `json:"items"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return types.MusicSearchResult{}, err
	}

	if len(response.Items) == 0 {
		return types.MusicSearchResult{}, fmt.Errorf("video not found")
	}

	item := response.Items[0]
	return types.MusicSearchResult{
		Title:      item.Snippet.Title,
		Artist:     item.Snippet.ChannelTitle,
		URL:        fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID),
		ID:         videoID,
		Duration:   item.ContentDetails.Duration,
		Thumbnail:  item.Snippet.Thumbnails.High.URL,
		SourceType: types.YouTube,
	}, nil
}

func GetSpotifyInfoByID(trackID string) (types.MusicSearchResult, error) {
	token, err := getSpotifyToken()
	if err != nil {
		return types.MusicSearchResult{}, err
	}

	apiURL := "https://api.spotify.com/v1/tracks/" + trackID

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return types.MusicSearchResult{}, err
	}

	req.Header.Add("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return types.MusicSearchResult{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return types.MusicSearchResult{}, err
	}

	var trackResponse struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Artists []struct {
			Name string `json:"name"`
		} `json:"artists"`
		Album struct {
			Images []struct {
				URL string `json:"url"`
			} `json:"images"`
		} `json:"album"`
		DurationMs   int `json:"duration_ms"`
		ExternalUrls struct {
			Spotify string `json:"spotify"`
		} `json:"external_urls"`
	}

	err = json.Unmarshal(body, &trackResponse)
	if err != nil {
		return types.MusicSearchResult{}, err
	}

	artistName := ""
	if len(trackResponse.Artists) > 0 {
		artistName = trackResponse.Artists[0].Name
	}

	thumbnailURL := ""
	if len(trackResponse.Album.Images) > 0 {
		thumbnailURL = trackResponse.Album.Images[0].URL
	}

	duration := fmt.Sprintf("%02d:%02d", trackResponse.DurationMs/60000, (trackResponse.DurationMs/1000)%60)

	return types.MusicSearchResult{
		Title:      trackResponse.Name,
		Artist:     artistName,
		URL:        trackResponse.ExternalUrls.Spotify,
		ID:         trackResponse.ID,
		Duration:   duration,
		Thumbnail:  thumbnailURL,
		SourceType: types.Spotify,
	}, nil
}

func GetYouTubeForSpotify(title, artist string) (types.MusicSearchResult, error) {
	query := fmt.Sprintf("%s %s", title, artist)

	results, err := SearchYouTube(query, 1)
	if err != nil {
		return types.MusicSearchResult{}, err
	}

	if len(results) == 0 {
		return types.MusicSearchResult{}, fmt.Errorf("no YouTube results found")
	}

	return results[0], nil
}

func GetYouTubeInfo(ytURL string) (types.MusicSearchResult, error) {
	var videoID string

	if strings.Contains(ytURL, "youtu.be") {
		parts := strings.Split(ytURL, "/")
		videoID = parts[len(parts)-1]
	} else if strings.Contains(ytURL, "youtube.com") {
		parsedURL, err := url.Parse(ytURL)
		if err != nil {
			return types.MusicSearchResult{}, err
		}

		query := parsedURL.Query()
		videoID = query.Get("v")
	}

	if videoID == "" {
		return types.MusicSearchResult{}, fmt.Errorf("could not extract video ID from URL")
	}

	return GetYouTubeInfoByID(videoID)
}

func GetSpotifyInfo(spotifyURL string) (types.MusicSearchResult, error) {
	var trackID string

	if strings.Contains(spotifyURL, "track") {
		parts := strings.Split(spotifyURL, "/")
		trackID = parts[len(parts)-1]

		// Remove any query parameters
		if strings.Contains(trackID, "?") {
			trackID = strings.Split(trackID, "?")[0]
		}
	} else {
		return types.MusicSearchResult{}, fmt.Errorf("URL must be a Spotify track URL")
	}

	if trackID == "" {
		return types.MusicSearchResult{}, fmt.Errorf("could not extract track ID from URL")
	}

	return GetSpotifyInfoByID(trackID)
}

func getSpotifyToken() (string, error) {
	clientID := config.Config.SpotifyClientId
	clientSecret := config.Config.SpotifyClientSecret

	tokenURL := "https://accounts.spotify.com/api/token"

	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(clientID, clientSecret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var tokenResponse struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}

	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return "", err
	}
	if tokenResponse.TokenType != "Bearer" {
		return "", fmt.Errorf("unexpected token type: %s", tokenResponse.TokenType)
	}

	return tokenResponse.AccessToken, nil
}
