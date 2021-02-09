package main

import (
	"encoding/json"
	"log"
	"net/http"
)

type mediaController interface {
	QueueTrack(url string)
	SkipCurrent()
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

	r.HandleFunc("/pleblist/clear", func(wr http.ResponseWriter, req *http.Request) {
	})

	r.HandleFunc("/pleblist/stop", func(wr http.ResponseWriter, req *http.Request) {
	})

	r.HandleFunc("/pleblist/skip", func(wr http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" {
			wr.WriteHeader(400)
			return
		}
		mc.SkipCurrent()
	})

	r.HandleFunc("/pleblist/add", func(wr http.ResponseWriter, req *http.Request) {
		if req.Method != "POST" {
			wr.WriteHeader(400)
			return
		}
		var item pleblistItem
		if err := json.NewDecoder(req.Body).Decode(&item); err != nil {
			wr.WriteHeader(400)
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
	controller := newMPVController()
	go controller.Background()
	pleblist("127.0.0.1:2044", controller)
}
