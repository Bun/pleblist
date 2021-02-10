package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
)

type mediaController interface {
	QueueTrack(url string)
	SkipCurrent()
	Playlist() []playlistItem
}

type playlistItem struct {
	ID    string `json:"id,omitempty"`
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
}

type pleblistItem struct {
	URL string `json:"url"`
}

// pleblist runs the pleblist API server that can be used to control media.
func pleblist(bind string, mc mediaController) {
	r := http.NewServeMux()

	// TODO: can we add a call to fetch the current playlist?
	// TODO: remove a specific track?

	r.HandleFunc("/", func(wr http.ResponseWriter, req *http.Request) {
		http.ServeFile(wr, req, "mgr.html")
	})

	r.HandleFunc("/pleblist", func(wr http.ResponseWriter, req *http.Request) {
		respondJSON(wr, mc.Playlist())
	})

	r.HandleFunc("/pleblist/current", func(wr http.ResponseWriter, req *http.Request) {
	})

	r.HandleFunc("/pleblist/clear", func(wr http.ResponseWriter, req *http.Request) {
	})

	r.HandleFunc("/pleblist/stop", func(wr http.ResponseWriter, req *http.Request) {
	})

	r.HandleFunc("/pleblist/skip", func(wr http.ResponseWriter, req *http.Request) {
		if !decodePOST(wr, req, nil) {
			// TODO: maybe arg to ensure we're skipping the right track
			return
		}
		mc.SkipCurrent()
	})

	r.HandleFunc("/pleblist/add", func(wr http.ResponseWriter, req *http.Request) {
		var item pleblistItem
		if !decodePOST(wr, req, &item) {
			return
		}
		mc.QueueTrack(item.URL)
	})

	s := http.Server{
		Handler: r,
		Addr:    bind,
	}
	if err := s.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func main() {
	bind := flag.String("bind", "127.0.0.1:2044", "Listen address for control server")
	flag.Parse()

	controller := newMPVController()
	go controller.Background()
	pleblist(*bind, controller)
}

func decodePOST(wr http.ResponseWriter, req *http.Request, obj interface{}) bool {
	if req.Method != "POST" {
		wr.WriteHeader(400)
		return false
	}
	if obj != nil {
		if err := json.NewDecoder(req.Body).Decode(&obj); err != nil {
			wr.WriteHeader(400)
			return false
		}
	}
	return true
}

func respondJSON(wr http.ResponseWriter, obj interface{}) {
	wr.Header().Set("Content-Type", "application/json")
	e := json.NewEncoder(wr)
	e.Encode(obj)
}
