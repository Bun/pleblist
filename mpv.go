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

// TODO: restart player on crash
// TODO: make window optional

type MPVCommand struct {
	Command   []string `json:"command"`
	Async     bool     `json:"async,omitempty"`
	RequestID int      `json:"request_id,omitempty"`
}

type MPVController struct {
	mu sync.Mutex
	e  *json.Encoder
}

func newMPVController() *MPVController {
	return &MPVController{}
}

func mpv(socket string) {
	cmd := exec.Command("mpv", "--no-terminal", "--no-taskbar-progress", "--image-display-duration=5", "--force-window=yes", "--input-ipc-server="+socket, "--idle",
		"--geometry=1280x720+0+0", "--autofit=1280x720") //, "--force-window-position")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// TODO: error handling
		log.Fatalln("mpv terminated:", err)
	}
}

// skip: playlist-next (unless it's the final track?)

func (c *MPVController) QueueTrack(url string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.e.Encode(MPVCommand{
		Command: []string{"loadfile", url, "append-play"},
		Async:   true,
	})
	if err != nil {
		log.Println("Failed to add URL", url, "->", err)
	}
	// Maybe:
	//must(e.Encode(MPVCommand{
	//	Command:   []string{"set", "pause", "no"},
	//	Async:     true,
	//	RequestID: 124,
	//}))
}

func (c *MPVController) SkipCurrent() {
	c.mu.Lock()
	defer c.mu.Unlock()
	err := c.e.Encode(MPVCommand{
		Command: []string{"playlist-next", "force"},
		Async:   true,
	})
	if err != nil {
		log.Println("Failed to skip track ->", err)
	}
}

func (c *MPVController) Background() {
	go mpv("mpv.socket")

	// TODO: automatically restore MPV if it exits

	var con net.Conn
	// Wait until socket becomes available
	for i := 0; i < 1000; i++ {
		var err error
		con, err = net.Dial("unix", "mpv.socket")
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
	c.mu.Unlock()

	d := json.NewDecoder(con)
	for {
		// TODO: do something with this? periodically clear playlist?
		var rm json.RawMessage
		if err := d.Decode(&rm); err != nil {
			log.Fatalln("Failed to decode:", err)
		}
		log.Printf("<< %v", string(rm))
	}
}
