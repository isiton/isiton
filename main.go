// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"log"
	"path"
	"strconv"

	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/elazarl/go-bindata-assetfs"

	"app/assets"
)

//go:generate go-bindata -o assets/bindata.go -pkg assets -nomemcopy public_html/...

// Serves index.html in case the requested file isn't found (or some other os.Stat error)
func serveIndex(serve http.Handler, fs assetfs.AssetFS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, err := fs.AssetInfo(path.Join(fs.Prefix, r.URL.Path))
		if err != nil {
			contents, err := fs.Asset(path.Join(fs.Prefix, "index.html"))
			if err != nil {
				http.Error(w, err.Error(), http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "text/html")
			w.Write(contents)
			return
		}
		serve.ServeHTTP(w, r)
	}
}

type Monitor struct {
	Hosts []Host `json:"hosts"`
}
type Host struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
	Addr  string `json:"addr"`

	PingResult

	Failures      int
	Count         int
	CountFailures int
}

type Latency struct {
	Min, Avg, Max float64
}

type PingResult struct {
	Online     bool
	PacketLoss int64
	Latency    Latency
}

func giveFloat(value string) float64 {
	result, _ := strconv.ParseFloat(value, 64)
	return result
}

func giveInt64(value string) int64 {
	result, _ := strconv.ParseInt(value, 10, 64)
	return result
}

func ParseConfig(filename string) (*Monitor, error) {
	config := &Monitor{}
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(contents, &config)
	for index, _ := range config.Hosts {
		config.Hosts[index].Index = index
	}
	return config, err
}

func main() {
	var (
		port   = flag.String("port", "8080", "Port for server")
		config = flag.String("config", "isiton.json", "config file name")
	)
	flag.Parse()

	log.SetFlags(0)

	monitor, err := ParseConfig(*config)
	if err != nil {
		log.Fatal("No hosts to monitor found: ", err)
	}

	hub := newHub()
	go hub.run()

	pdp := NewPinger()

	for key, _ := range monitor.Hosts {
		go func(host *Host) {
			for {
				host.PingResult = pdp.RunPing(host.Addr)
				host.Count++
				if host.Online {
					host.Failures = 0
				} else {
					host.Failures++
					host.CountFailures++
				}
				hub.broadcast <- *host
			}
		}(&monitor.Hosts[key])
	}

	http.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	assetPrefix := "public_html"
	assets := assetfs.AssetFS{
		assets.Asset,
		assets.AssetDir,
		assets.AssetInfo,
		assetPrefix,
	}
	server := http.FileServer(&assets)

	http.HandleFunc("/", serveIndex(server, assets))

	log.Println("Started listening on port", *port)
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
