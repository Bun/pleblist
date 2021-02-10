package main

import (
	"encoding/json"
	"log"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"
)

// TODO: /tmp was more convenient when running ./pleblist over a weird sshfs
// mount
var mpvSocket = "/tmp/pleblist.mpv.socket"

// TODO: restart player on crash
// TODO: make window optional

// TODO: clean up playlist

type MPVCommand struct {
	Command   []interface{} `json:"command"`
	RequestID int           `json:"request_id,omitempty"`
}

type mpvController struct {
	mu       sync.Mutex
	e        *json.Encoder
	playlist []playlistItem
}

func newMPVController() *mpvController {
	return &mpvController{}
}

func mpv(socket string) {
	cmd := exec.Command("mpv",
		"--no-terminal", "--no-taskbar-progress",
		"--osc=no", // Disables OSC but also disables the idle "drop file here" screen
		"--image-display-duration=5",
		"--force-window=yes",
		"--input-ipc-server="+socket, "--idle",
		"--geometry=1280x720+0+0", "--autofit=1280x720") //, "--force-window-position")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	closeChild(cmd)
	if err := cmd.Run(); err != nil {
		// TODO: error handling
		log.Fatalln("mpv terminated:", err)
	}
}

// skip: playlist-next (unless it's the final track?)

func (c *mpvController) Playlist() (pis []playlistItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	pis = make([]playlistItem, len(c.playlist))
	for i, v := range c.playlist {
		pis[i] = v
	}
	return
}

func (c *mpvController) QueueTrack(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.e.Encode(MPVCommand{
		Command: []interface{}{"loadfile", url, "append-play"},
	})
	if err != nil {
		log.Println("Failed to add URL", url, "->", err)
	}
}

func (c *mpvController) SkipCurrent() {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.e.Encode(MPVCommand{
		Command: []interface{}{"playlist-remove", "current"},
	})
	if err != nil {
		log.Println("Failed to skip track ->", err)
	}
}

func (c *mpvController) updatePlaylist(pc []mpvPlaylistEntry) {
	var playlist []playlistItem
	skip := true
	for _, v := range pc {
		if v.Current {
			skip = false
		}
		if !skip {
			playlist = append(playlist, playlistItem{
				URL: v.Filename,
			})
		}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.playlist = playlist
}

func (c *mpvController) removeHead() {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.e.Encode(MPVCommand{
		Command: []interface{}{"playlist-remove", "0"},
	})
	if err != nil {
		log.Println("Failed to remove playlist track 0 ->", err)
	}
}

func (c *mpvController) Background() {
	go mpv(mpvSocket)

	// TODO: automatically restore MPV if it exits

	var con net.Conn
	// Wait until socket becomes available
	for i := 0; i < 1000; i++ {
		var err error
		con, err = net.Dial("unix", mpvSocket)
		if err == nil {
			log.Println("Connected to mpv socket", i)
			break
		}
		time.Sleep(time.Millisecond * 10)
	}
	if con == nil {
		log.Fatal("Failed to connect to MPV")
	}

	c.mu.Lock()
	c.e = json.NewEncoder(con)
	c.e.Encode(MPVCommand{Command: []interface{}{"observe_property", 1, "playlist"}})
	c.e.Encode(MPVCommand{Command: []interface{}{"observe_property", 2, "media-title"}})
	c.e.Encode(MPVCommand{Command: []interface{}{"observe_property", 3, "duration"}})
	// playtime-remaining ?
	c.mu.Unlock()

	d := json.NewDecoder(con)
	for {
		// FIXME
		var rm json.RawMessage
		if err := d.Decode(&rm); err != nil {
			log.Fatalln("Failed to decode:", err)
		}
		log.Printf("<< %v", string(rm))

		var m mpvMessage
		json.Unmarshal([]byte(rm), &m)

		switch m.Event {
		case "start-file":
			// New song loading

		case "end-file":
			c.removeHead()

		case "property-change":
			if m.Name == "playlist" {
				var pc mpvPlaylistPC
				json.Unmarshal([]byte(rm), &pc)
				c.updatePlaylist(pc.Data)
			}
		}
	}
}

type (
	mpvMessage struct {
		RequestID int    `json:"request_id"`
		Error     string `json:"error"`
		Event     string `json:"event"`

		// property-change
		ID   int64           `json:"id"`
		Name string          `json:"name"`
		Data json.RawMessage `json:"data"`
	}

	mpvPlaylistPC struct {
		Data []mpvPlaylistEntry `json:"data"`
	}

	mpvPlaylistEntry struct {
		Filename string `json:"filename"`
		Current  bool   `json:"current"`
		Playing  bool   `json:"playing"`
	}
)
