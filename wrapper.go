package main

import (
	"encoding/binary"
	"github.com/bwmarrin/discordgo"
	"github.com/bwmarrin/lit"
	"io"
	"os/exec"
	"runtime"
	"strings"
)

// Wrapper function for playing songs
func play(s *discordgo.Session, song string, i *discordgo.Interaction, voiceChannel, guild, username string, random bool) {
	switch {
	case strings.HasPrefix(song, "spotify:playlist:"):
		spotifyPlaylist(s, guild, voiceChannel, username, song, i, random)

	case isValidURL(song):
		downloadAndPlay(s, guild, voiceChannel, song, username, i, random)

	default:
		searchDownloadAndPlay(s, guild, voiceChannel, song, username, i, random)
	}
}

// Wrapper function for playSound, also waits for the song to finish to download and then closes it's pipe
func playSoundStream(s *discordgo.Session, guildID, channelID, fileName string, i *discordgo.Interaction, stdout io.ReadCloser, cmds []*exec.Cmd) {
	deleteInteraction(s, i, nil)

	playSound(s, guildID, channelID, fileName, i, stdout, nil, cmds)

	switch runtime.GOOS {
	case "windows":
		_ = stdout.Close()

	// TODO: On linux, if we close the pipe, tee just quits without waiting for the song to completely download
	default:
		if stdout == nil {
			break
		}

		var err error
		var opuslen int16

		for {
			// Read opus frame length from dca file.
			err = binary.Read(stdout, binary.LittleEndian, &opuslen)

			// If this is the end of the file, just return.
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}

			if err != nil {
				lit.Error("Error reading from dca file: %s", err)
				break
			}

			// Read encoded pcm from dca file.
			InBuf := make([]byte, opuslen)
			_ = binary.Read(stdout, binary.LittleEndian, &InBuf)
		}
	}

	cmdsWait(cmds)

	server[guildID].stream.Unlock()
}
