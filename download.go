package main

import (
	"encoding/json"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"github.com/zmb3/spotify"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

//Download and plays a song from a youtube link
func downloadAndPlay(s *discordgo.Session, guildID, channelID, link, user, txtChannel string) {
	go sendAndDeleteEmbed(s, NewEmbed().SetTitle(s.State.User.Username).AddField("Enqueued", link).SetColor(0x7289DA).MessageEmbed, txtChannel)

	//Check if the song is the db, to speedup things
	el := checkInDb(link)
	if el.title != "" {
		el.user = user
		queue[guildID] = append(queue[guildID], el)
		go playSound(s, guildID, channelID, el.id+".dca", txtChannel, findQueuePointer(guildID, el.id))
		return
	}

	//Gets info about songs
	out, _ := exec.Command("youtube-dl", "--ignore-errors", "-q", "--no-warnings", "-j", link).Output()

	//Parse output as string, splitting it on every newline
	strOut := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")

	var ytdl YoutubeDL
	//We parse every track as individual json, because youtube-dl
	for _, singleJson := range strOut {
		_ = json.Unmarshal([]byte(singleJson), &ytdl)
		fileName := ytdl.ID + "-" + ytdl.Extractor
		el := Queue{ytdl.Title, formatDuration(ytdl.Duration), fileName, ytdl.WebpageURL, user, nil, 0, ""}

		//Checks if video is already downloaded
		_, err := os.Stat("./audio_cache/" + fileName + ".dca")

		//We add the song to the db, for faster parsing
		addToDb(el)

		//If we have a single song, we also add it with the given link
		if len(strOut) == 1 {
			el.link = link
			addToDb(el)
		}

		//If not, we download and convert it
		if err != nil {
			//Download
			_ = exec.Command("youtube-dl", "-o", "download/"+fileName+".m4a", "-x", "--audio-format", "m4a", ytdl.WebpageURL).Run()

			//Conversion to DCA
			switch runtime.GOOS {
			case "linux":
				_ = exec.Command("bash", "gen.sh", fileName, fileName+".m4a").Run()
				break
			case "windows":
				_ = exec.Command("gen.bat", fileName, fileName+".m4a").Run()
			}

			err = os.Remove("./download/" + fileName + ".m4a")
		}

		queue[guildID] = append(queue[guildID], el)
		go playSound(s, guildID, channelID, fileName+".dca", txtChannel, findQueuePointer(guildID, fileName))

	}

}

//Searches a song from the query on youtube
func searchDownloadAndPlay(s *discordgo.Session, guildID, channelID, query, user, txtChannel string) {
	//Gets video id
	out, _ := exec.Command("youtube-dl", "--get-id", "ytsearch:\""+query+"\"").Output()
	ids := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")

	//Calls download and play for every id we get
	for _, id := range ids {
		downloadAndPlay(s, guildID, channelID, "https://www.youtube.com/watch?v="+id, user, txtChannel)
	}

}

//Enqueues song from a spotify playlist, searching them on youtube
func spotifyPlaylist(s *discordgo.Session, guildID, channelID, user, playlistId, txtChannel string) {

	//We get the playlist from it's link
	playlist, err := client.GetPlaylist(spotify.ID(strings.TrimPrefix(playlistId, "spotify:playlist:")))
	if err != nil {
		fmt.Println(err)
		return
	}

	//We parse every single song, searching it on youtube
	for _, track := range playlist.Tracks.Tracks {
		go searchDownloadAndPlay(s, guildID, channelID, track.Track.Name+" - "+track.Track.Artists[0].Name, user, txtChannel)
	}

}
