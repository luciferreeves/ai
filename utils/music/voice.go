package music

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"layeh.com/gopus"
)

const (
	channels  int = 2                   // 1 for mono, 2 for stereo
	frameRate int = 48000               // audio sampling rate
	frameSize int = 960                 // size of each audio frame
	maxBytes  int = (frameSize * 2) * 2 // max size of opus data
)

// VoiceInstance represents a voice connection
type VoiceInstance struct {
	GuildID        string
	ChannelID      string
	Connection     *discordgo.VoiceConnection
	Playing        bool
	StopChannel    chan bool
	OpusEncoder    *gopus.Encoder
	mu             sync.Mutex
	CurrentTrackID string
}

var (
	VoiceConnection = make(map[string]*VoiceInstance)
	VoiceMutex      = &sync.Mutex{}
)

// Stop stops the current playback
func (v *VoiceInstance) Stop() {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.Playing {
		select {
		case v.StopChannel <- true:
			// Signal sent here
		default:
			// Channel already has a signal
		}
		close(v.StopChannel)
		v.StopChannel = make(chan bool, 1)
		v.Playing = false
	}
}

// JoinVoiceChannel makes the bot join a voice channel
func JoinVoiceChannel(s *discordgo.Session, guildID, channelID string) (*VoiceInstance, error) {
	VoiceMutex.Lock()
	defer VoiceMutex.Unlock()

	// Check if already in a voice channel in this guild
	if voice, exists := VoiceConnection[guildID]; exists {
		return voice, nil
	}

	// not in a voice channel, create a new one
	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return nil, fmt.Errorf("failed to join voice channel: %w", err)
	}

	encoder, err := gopus.NewEncoder(frameRate, channels, gopus.Audio)
	if err != nil {
		vc.Disconnect()
		return nil, fmt.Errorf("failed to create opus encoder: %w", err)
	}

	voiceInstance := &VoiceInstance{
		GuildID:     guildID,
		ChannelID:   channelID,
		Connection:  vc,
		Playing:     false,
		StopChannel: make(chan bool, 1),
		OpusEncoder: encoder,
	}

	VoiceConnection[guildID] = voiceInstance
	return voiceInstance, nil
}

// LeaveVoiceChannel makes the bot leave a voice channel
func LeaveVoiceChannel(guildID string) error {
	VoiceMutex.Lock()
	defer VoiceMutex.Unlock()

	voice, exists := VoiceConnection[guildID]
	if !exists {
		return fmt.Errorf("not in a voice channel")
	}

	// Stop current playback
	voice.Stop()

	// Disconnect
	err := voice.Connection.Disconnect()
	if err != nil {
		return fmt.Errorf("failed to disconnect from voice channel: %w", err)
	}

	// remove from map
	delete(VoiceConnection, guildID)
	return nil
}

// IsUserInSameVC checks if the user is in the same voice channel as the bot
func IsUserInSameVC(s *discordgo.Session, guildID, userID string) (bool, string) {
	// Voice state
	guild, err := s.State.Guild(guildID)
	if err != nil {
		return false, ""
	}

	var userChannelID string
	for _, vs := range guild.VoiceStates {
		if vs.UserID == userID {
			userChannelID = vs.ChannelID
			break
		}
	}

	if userChannelID == "" {
		return false, "" // user not in a voice channel
	}

	// Check if bot is in a voice channel
	VoiceMutex.Lock()
	defer VoiceMutex.Unlock()

	voice, exists := VoiceConnection[guildID]
	if !exists {
		return true, userChannelID // bot not in a voice channel, but no conflict
	}

	return voice.ChannelID == userChannelID, userChannelID
}

// PlayYouTube downloads and plays a YouTube video
func (v *VoiceInstance) PlayYouTube(videoURL, videoID string) error {
	fmt.Printf("Starting to play: %s (ID: %s)\n", videoURL, videoID)

	// Create a new stop channel for this playback
	var oldStopChan chan bool

	v.mu.Lock()
	// If already playing, properly stop the previous playback
	if v.Playing {
		fmt.Println("Stopping current playback before starting new one...")
		// Save the old channel to send the stop signal after we release the lock
		oldStopChan = v.StopChannel
		// Create a new channel for the new playback
		v.StopChannel = make(chan bool, 1)
	} else {
		v.StopChannel = make(chan bool, 1)
	}

	v.Playing = true
	v.CurrentTrackID = videoID
	stopChan := v.StopChannel
	v.mu.Unlock()

	// Send stop signal to old channel if it exists
	// Do this outside the lock to avoid deadlock
	if oldStopChan != nil {
		// Signal the old playback to stop
		select {
		case oldStopChan <- true:
			fmt.Println("Stop signal sent to previous playback")
		default:
			fmt.Println("Could not send stop signal, channel might be full or closed")
		}
		// Wait a moment for the previous playback to clean up
		time.Sleep(100 * time.Millisecond)
	}

	// Ensure temp directory exists
	err := os.MkdirAll("./temp", 0755)
	if err != nil {
		fmt.Printf("Error creating temp directory: %v\n", err)
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Create a unique filename
	fileName := fmt.Sprintf("./temp/%s_%d.mp3", videoID, time.Now().Unix())
	fmt.Printf("Downloading to: %s\n", fileName)

	// Use yt-dlp to download audio
	downloadCmd := exec.Command("yt-dlp", "-x", "--audio-format", "mp3",
		"--audio-quality", "0", "--no-playlist", "--output", fileName, videoURL)

	// Set up pipes to capture output for debugging
	downloadCmd.Stdout = os.Stdout
	downloadCmd.Stderr = os.Stderr

	fmt.Println("Starting download...")
	err = downloadCmd.Run()
	if err != nil {
		fmt.Printf("Download error: %v\n", err)
		v.mu.Lock()
		v.Playing = false
		v.mu.Unlock()
		return fmt.Errorf("failed to download audio: %w", err)
	}

	fmt.Printf("Download complete, starting playback\n")

	// Check if file exists and get its size
	fileInfo, err := os.Stat(fileName)
	if err != nil {
		fmt.Printf("File stat error: %v\n", err)
		v.mu.Lock()
		v.Playing = false
		v.mu.Unlock()
		return fmt.Errorf("file stat error: %w", err)
	}
	fmt.Printf("File size: %d bytes\n", fileInfo.Size())

	// Ensure file gets deleted after playback
	defer os.Remove(fileName)

	// Make sure we're not already in a speaking state
	v.Connection.Speaking(false)
	time.Sleep(50 * time.Millisecond)

	// Set speaking status
	err = v.Connection.Speaking(true)
	if err != nil {
		fmt.Printf("Speaking error: %v\n", err)
		v.mu.Lock()
		v.Playing = false
		v.mu.Unlock()
		return fmt.Errorf("speaking error: %w", err)
	}
	defer v.Connection.Speaking(false)

	// Use ffmpeg for playback
	ffmpeg := exec.Command("ffmpeg", "-i", fileName, "-f", "s16le", "-ar", "48000", "-ac", "2", "pipe:1")
	ffmpegout, err := ffmpeg.StdoutPipe()
	if err != nil {
		fmt.Printf("FFmpeg pipe error: %v\n", err)
		return fmt.Errorf("ffmpeg pipe error: %w", err)
	}

	ffmpeg.Stderr = os.Stderr
	err = ffmpeg.Start()
	if err != nil {
		fmt.Printf("FFmpeg start error: %v\n", err)
		return fmt.Errorf("ffmpeg start error: %w", err)
	}

	// Store ffmpeg process for proper cleanup
	ffmpegProcess := ffmpeg.Process
	defer func() {
		ffmpegProcess.Kill()
		ffmpeg.Wait() // Wait for the process to exit to avoid zombies
	}()

	// Read and send loop
	buf := make([]int16, frameSize*channels)

	playbackDone := make(chan error, 1)
	go func() {
		for {
			// Read data from ffmpeg
			err = binary.Read(ffmpegout, binary.LittleEndian, &buf)
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				playbackDone <- nil
				return
			}
			if err != nil {
				playbackDone <- fmt.Errorf("error reading from ffmpeg: %w", err)
				return
			}

			// Encode with opus
			opus, err := v.OpusEncoder.Encode(buf, frameSize, maxBytes)
			if err != nil {
				playbackDone <- fmt.Errorf("opus encoding error: %w", err)
				return
			}

			// Send to Discord
			select {
			case v.Connection.OpusSend <- opus:
				// Sent successfully
			case <-stopChan:
				playbackDone <- nil
				return
			}
		}
	}()

	// Wait for playback to finish or stop signal
	select {
	case err := <-playbackDone:
		if err != nil {
			fmt.Printf("Playback error: %v\n", err)
		} else {
			fmt.Println("Playback completed normally")
		}
	case <-stopChan:
		fmt.Println("Playback stopped by request")
	}

	// Make sure to kill ffmpeg
	ffmpegProcess.Kill()

	v.mu.Lock()
	v.Playing = false
	v.mu.Unlock()

	return nil
}

// func (v *VoiceInstance) playAudioFile(filename string, stopChan chan bool) error {
// 	// Start ffmpeg to convert the file to PCM
// 	ffmpeg := exec.Command("ffmpeg", "-i", filename, "-f", "s16le", "-ar", "48000", "-ac", "2", "pipe:1")
// 	ffmpegout, err := ffmpeg.StdoutPipe()
// 	if err != nil {
// 		return fmt.Errorf("ffmpeg stdout error: %w", err)
// 	}

// 	ffmpeg.Stderr = os.Stderr
// 	err = ffmpeg.Start()
// 	if err != nil {
// 		return fmt.Errorf("ffmpeg start error: %w", err)
// 	}

// 	// Make sure to kill ffmpeg when we're done
// 	defer ffmpeg.Process.Kill()

// 	// Set speaking status
// 	err = v.Connection.Speaking(true)
// 	if err != nil {
// 		return fmt.Errorf("speaking error: %w", err)
// 	}
// 	defer v.Connection.Speaking(false)

// 	// Create a buffer for reading from ffmpeg
// 	buf := make([]int16, frameSize*channels)

// 	// Read and send loop
// 	for {
// 		// Check if we've been asked to stop
// 		select {
// 		case <-stopChan:
// 			return nil
// 		default:
// 			// Continue playing
// 		}

// 		// Read data from ffmpeg
// 		err = binary.Read(ffmpegout, binary.LittleEndian, &buf)
// 		if err == io.EOF || err == io.ErrUnexpectedEOF {
// 			// End of file
// 			return nil
// 		}
// 		if err != nil {
// 			return fmt.Errorf("error reading from ffmpeg: %w", err)
// 		}

// 		// Encode with opus
// 		opus, err := v.OpusEncoder.Encode(buf, frameSize, maxBytes)
// 		if err != nil {
// 			return fmt.Errorf("opus encoding error: %w", err)
// 		}

// 		// Send to Discord
// 		select {
// 		case v.Connection.OpusSend <- opus:
// 			// Sent successfully
// 		case <-stopChan:
// 			return nil
// 		}
// 	}
// }
