package main

//go:generate go-bindata -o ../../assets/bindata.go -pkg assets -nomemcopy -prefix ../.. ../../public_html/...

import (
	"flag"
	"log"
	"path"

	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/elazarl/go-bindata-assetfs"

	"github.com/isiton/isiton/assets"
	"github.com/isiton/isiton/pinger"
)

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
	Settings Settings `json:"settings"`
	Hosts    []Host   `json:"hosts"`
}

type Settings struct {
	Warning int `json:"warning"`
}

type Host struct {
	Index int    `json:"index"`
	Name  string `json:"name"`
	Addr  string `json:"addr"`

	pinger.Result

	Failures      int
	Count         int
	CountFailures int
	Warning       int
}

func ParseConfig(filename string) (*Monitor, error) {
	config := &Monitor{}
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(contents, &config)
	if config.Settings.Warning == 0 {
		config.Settings.Warning = 30
	}
	for index, _ := range config.Hosts {
		config.Hosts[index].Index = index
		if config.Hosts[index].Warning == 0 {
			config.Hosts[index].Warning = config.Settings.Warning
		}
	}
	return config, err
}

func main() {
	var (
		port   = flag.String("port", "8080", "Port for server")
		config = flag.String("config", "isiton.json", "config file name")
		times  = flag.Int("ping-times", 4, "Number of pings per cycle")
		delay  = flag.Int("ping-delay", 1, "Delay after each ping (seconds)")
	)
	flag.Parse()

	log.SetFlags(0)

	monitor, err := ParseConfig(*config)
	if err != nil {
		log.Fatal("No hosts to monitor found: ", err)
	}

	hub := newHub()
	go hub.run()

	pdp := pinger.New()

	for key, _ := range monitor.Hosts {
		go func(host *Host) {
			for {
				host.Result = pdp.Ping(host.Addr, *times, *delay)
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
