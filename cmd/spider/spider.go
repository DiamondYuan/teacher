package main

import (
	"github.com/GeorgeYuen/dht"
	"net/http"
	_ "net/http/pprof"
	"encoding/hex"
	"encoding/json"
	"github.com/GeorgeYuen/teacher/sql"
)

type file struct {
	Path   []interface{} `json:"path"`
	Length int           `json:"length"`
}

type bitTorrent struct {
	InfoHash string `json:"infohash"`
	Name     string `json:"name"`
	Files    []file `json:"files,omitempty"`
	Length   int    `json:"length,omitempty"`
}

func main() {
	go func() {
		http.ListenAndServe(":6060", nil)
	}()
	w := dht.NewWire(65536, 1024, 256)
	go func() {
		for resp := range w.Response() {
			metadata, err := dht.Decode(resp.MetadataInfo)
			if err != nil {
				continue
			}
			info := metadata.(map[string]interface{})

			if _, ok := info["name"]; !ok {
				continue
			}

			bt := bitTorrent{
				InfoHash: hex.EncodeToString(resp.InfoHash),
				Name:     info["name"].(string),
			}

			if v, ok := info["files"]; ok {
				files := v.([]interface{})
				bt.Files = make([]file, len(files))

				for i, item := range files {
					f := item.(map[string]interface{})
					bt.Files[i] = file{
						Path:   f["path"].([]interface{}),
						Length: f["length"].(int),
					}
				}
			} else if _, ok := info["length"]; ok {
				bt.Length = info["length"].(int)
			}

			data, err := json.Marshal(bt)
			if err == nil {
				tracherSql.SaveData(resp.InfoHash, data, resp.MetadataInfo)
			}
		}
	}()
	go w.Run()

	config := dht.NewCrawlConfig()
	config.OnAnnouncePeer = func(infoHash, ip string, port int) {
		if len(infoHash) == 20 {
			InfoHash := hex.EncodeToString([]byte(infoHash))
			tracherSql.SaveHash(InfoHash, ip)
		}
		w.Request([]byte(infoHash), ip, port)
	}
	d := dht.New(config)
	d.Run()
}
