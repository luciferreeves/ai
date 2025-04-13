package music

import (
	"ai/types"
	"ai/utils/logger"
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
	channels  int = 2
	frameRate int = 48000
	frameSize int = 960
	maxBytes  int = (frameSize * 2) * 2
)

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

func (v *VoiceInstance) Stop() {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.Playing {
		select {
		case v.StopChannel <- true:
		default:
		}
		close(v.StopChannel)
		v.StopChannel = make(chan bool, 1)
		v.Playing = false
	}
}

func JoinVoiceChannel(s *discordgo.Session, guildID, channelID string) (*VoiceInstance, error) {
	VoiceMutex.Lock()
	defer VoiceMutex.Unlock()

	if voice, exists := VoiceConnection[guildID]; exists {
		return voice, nil
	}

	vc, err := s.ChannelVoiceJoin(guildID, channelID, false, true)
	if err != nil {
		return nil, err
	}

	encoder, err := gopus.NewEncoder(frameRate, channels, gopus.Audio)
	if err != nil {
		vc.Disconnect()
		return nil, err
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

func LeaveVoiceChannel(guildID string) error {
	VoiceMutex.Lock()
	defer VoiceMutex.Unlock()

	voice, exists := VoiceConnection[guildID]
	if !exists {
		return nil
	}

	voice.Stop()

	err := voice.Connection.Disconnect()
	if err != nil {
		return err
	}

	delete(VoiceConnection, guildID)
	return nil
}

func IsUserInSameVC(s *discordgo.Session, guildID, userID string) (bool, string) {
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
		return false, ""
	}

	VoiceMutex.Lock()
	defer VoiceMutex.Unlock()

	voice, exists := VoiceConnection[guildID]
	if !exists {
		return true, userChannelID
	}

	return voice.ChannelID == userChannelID, userChannelID
}

func (v *VoiceInstance) PlayYouTube(videoURL, videoID string) error {
	logger.Log("Starting to play: "+videoURL, types.LogOptions{
		Prefix: "Music Player",
		Level:  types.Info,
	})

	var oldStopChan chan bool

	v.mu.Lock()
	if v.Playing {
		logger.Log("Stopping current playback", types.LogOptions{
			Prefix: "Music Player",
			Level:  types.Info,
		})
		oldStopChan = v.StopChannel
		v.StopChannel = make(chan bool, 1)
	} else {
		v.StopChannel = make(chan bool, 1)
	}

	v.Playing = true
	v.CurrentTrackID = videoID
	stopChan := v.StopChannel
	v.mu.Unlock()

	if oldStopChan != nil {
		select {
		case oldStopChan <- true:
			logger.Log("Stop signal sent to previous playback", types.LogOptions{
				Prefix: "Music Player",
				Level:  types.Debug,
			})
		default:
			logger.Log("Could not send stop signal", types.LogOptions{
				Prefix: "Music Player",
				Level:  types.Debug,
			})
		}
		time.Sleep(100 * time.Millisecond)
	}

	err := os.MkdirAll("./temp", 0755)
	if err != nil {
		logger.Log("Failed to create temp directory: "+err.Error(), types.LogOptions{
			Prefix: "Music Player",
			Level:  types.Error,
		})
		return err
	}

	fileName := fmt.Sprintf("./temp/%s_%d.mp3", videoID, time.Now().Unix())
	logger.Log("Downloading to: "+fileName, types.LogOptions{
		Prefix: "Music Player",
		Level:  types.Debug,
	})

	var downloadCmd *exec.Cmd
	cookiesFile := "./cookies/cookies.txt"
	if _, err := os.Stat(cookiesFile); err == nil {
		logger.Log("Using cookies file: "+cookiesFile, types.LogOptions{
			Prefix: "Music Player",
			Level:  types.Debug,
		})
		// log the cookies file data
		cookiesData, err := os.ReadFile(cookiesFile)
		if err != nil {
			logger.Log("Error reading cookies file: "+err.Error(), types.LogOptions{
				Prefix: "Music Player",
				Level:  types.Error,
			})
		}

		logger.Log("Cookies file data: "+string(cookiesData), types.LogOptions{
			Prefix: "Music Player",
			Level:  types.Debug,
		})
		downloadCmd = exec.Command("yt-dlp", "--no-warnings", "--quiet", "-x", "--audio-format", "mp3",
			"--audio-quality", "0", "--no-playlist", "--cookies", cookiesFile, "--output", fileName, videoURL)
	} else {
		logger.Log("No cookies file found, downloading without cookies", types.LogOptions{
			Prefix: "Music Player",
			Level:  types.Debug,
		})
		downloadCmd = exec.Command("yt-dlp", "--no-warnings", "--quiet", "-x", "--audio-format", "mp3",
			"--audio-quality", "0", "--no-playlist", "--output", fileName, videoURL)
	}

	// Create logs for stdout and stderr to capture yt-dlp output
	stdout, err := downloadCmd.StdoutPipe()
	if err != nil {
		logger.Log("Error creating StdoutPipe: "+err.Error(), types.LogOptions{
			Prefix: "Music Player",
			Level:  types.Error,
		})
		return err
	}
	stderr, err := downloadCmd.StderrPipe()
	if err != nil {
		logger.Log("Error creating StderrPipe: "+err.Error(), types.LogOptions{
			Prefix: "Music Player",
			Level:  types.Error,
		})
		return err
	}

	// Start the download process
	err = downloadCmd.Start()
	if err != nil {
		logger.Log("Error starting yt-dlp command: "+err.Error(), types.LogOptions{
			Prefix: "Music Player",
			Level:  types.Error,
		})
		return err
	}

	// Log the stdout and stderr
	go func() {
		stdoutLogs := make([]byte, 1024)
		for {
			n, err := stdout.Read(stdoutLogs)
			if err != nil {
				break
			}
			logger.Log("yt-dlp stdout: "+string(stdoutLogs[:n]), types.LogOptions{
				Prefix: "Music Player",
				Level:  types.Debug,
			})
		}
	}()
	go func() {
		stderrLogs := make([]byte, 1024)
		for {
			n, err := stderr.Read(stderrLogs)
			if err != nil {
				break
			}
			logger.Log("yt-dlp stderr: "+string(stderrLogs[:n]), types.LogOptions{
				Prefix: "Music Player",
				Level:  types.Error,
			})
		}
	}()

	err = downloadCmd.Wait()
	if err != nil {
		logger.Log("Download error: "+err.Error(), types.LogOptions{
			Prefix: "Music Player",
			Level:  types.Error,
		})
		v.mu.Lock()
		v.Playing = false
		v.mu.Unlock()
		return err
	}

	logger.Log("Download complete, starting playback", types.LogOptions{
		Prefix: "Music Player",
		Level:  types.Info,
	})

	fileInfo, err := os.Stat(fileName)
	if err != nil {
		logger.Log("File stat error: "+err.Error(), types.LogOptions{
			Prefix: "Music Player",
			Level:  types.Error,
		})
		v.mu.Lock()
		v.Playing = false
		v.mu.Unlock()
		return err
	}

	logger.Log(fmt.Sprintf("File size: %d bytes", fileInfo.Size()), types.LogOptions{
		Prefix: "Music Player",
		Level:  types.Debug,
	})

	defer os.Remove(fileName)

	err = v.playAudioFile(fileName, stopChan)
	if err != nil {
		logger.Log("Playback error: "+err.Error(), types.LogOptions{
			Prefix: "Music Player",
			Level:  types.Error,
		})
	}

	v.mu.Lock()
	v.Playing = false
	v.mu.Unlock()

	return err
}

func (v *VoiceInstance) playAudioFile(filename string, stopChan chan bool) error {
	v.Connection.Speaking(false)
	time.Sleep(50 * time.Millisecond)

	err := v.Connection.Speaking(true)
	if err != nil {
		logger.Log("Speaking error: "+err.Error(), types.LogOptions{
			Prefix: "Music Player",
			Level:  types.Error,
		})
		return err
	}
	defer v.Connection.Speaking(false)

	ffmpeg := exec.Command("ffmpeg", "-hide_banner", "-loglevel", "quiet", "-i", filename, "-f", "s16le", "-ar", "48000", "-ac", "2", "pipe:1")
	ffmpegout, err := ffmpeg.StdoutPipe()
	if err != nil {
		logger.Log("FFmpeg pipe error: "+err.Error(), types.LogOptions{
			Prefix: "Music Player",
			Level:  types.Error,
		})
		return err
	}

	ffmpeg.Stderr = nil
	err = ffmpeg.Start()
	if err != nil {
		logger.Log("FFmpeg start error: "+err.Error(), types.LogOptions{
			Prefix: "Music Player",
			Level:  types.Error,
		})
		return err
	}

	ffmpegProcess := ffmpeg.Process
	defer func() {
		ffmpegProcess.Kill()
		ffmpeg.Wait()
	}()

	buf := make([]int16, frameSize*channels)

	playbackDone := make(chan error, 1)
	go func() {
		for {
			err = binary.Read(ffmpegout, binary.LittleEndian, &buf)
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				playbackDone <- nil
				return
			}
			if err != nil {
				playbackDone <- err
				return
			}

			opus, err := v.OpusEncoder.Encode(buf, frameSize, maxBytes)
			if err != nil {
				playbackDone <- err
				return
			}

			select {
			case v.Connection.OpusSend <- opus:
			case <-stopChan:
				playbackDone <- nil
				return
			}
		}
	}()

	select {
	case err := <-playbackDone:
		if err != nil {
			logger.Log("Playback error: "+err.Error(), types.LogOptions{
				Prefix: "Music Player",
				Level:  types.Error,
			})
		} else {
			logger.Log("Playback completed", types.LogOptions{
				Prefix: "Music Player",
				Level:  types.Success,
			})
		}
		return err
	case <-stopChan:
		logger.Log("Playback stopped by request", types.LogOptions{
			Prefix: "Music Player",
			Level:  types.Info,
		})
		return nil
	}
}
