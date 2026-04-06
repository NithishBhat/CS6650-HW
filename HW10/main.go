package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
)

// each key maps to a value + version number
type Entry struct {
	Value   string `json:"value"`
	Version int    `json:"version"`
}

// the actual kv store, just a map behind a mutex
var (
	store   = make(map[string]Entry)
	storeMu sync.RWMutex
)

// all of these get set from env vars in initConfig()
var (
	role       string   // "leader", "follower", or "leaderless"
	nodeID     int
	peers      []string // everyone in the cluster, including us
	leaderAddr string
	W, R       int
)

func initConfig() {
	role = os.Getenv("ROLE")
	if role == "" {
		role = "leader"
	}
	nodeID, _ = strconv.Atoi(os.Getenv("NODE_ID"))

	if p := os.Getenv("PEERS"); p != "" {
		peers = strings.Split(p, ",")
	}
	leaderAddr = os.Getenv("LEADER_ADDR")

	W, _ = strconv.Atoi(os.Getenv("W"))
	if W == 0 {
		W = 5
	}
	R, _ = strconv.Atoi(os.Getenv("R"))
	if R == 0 {
		R = 1
	}

	log.Printf("Node %d starting: role=%s W=%d R=%d peers=%v", nodeID, role, W, R, peers)
}

// get everyone else's address (skip ourselves)
func otherPeers() []string {
	var others []string
	for i, p := range peers {
		if i != nodeID {
			others = append(others, p)
		}
	}
	return others
}

// POST /set - client sends {"key":"k","value":"v"}, we store it and replicate
func handleSet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Key == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if role == "follower" {
		http.Error(w, "writes go to leader", http.StatusBadRequest)
		return
	}

	// bump version, save locally first
	storeMu.Lock()
	e := store[req.Key]
	e.Version++
	e.Value = req.Value
	store[req.Key] = e
	version := e.Version
	storeMu.Unlock()

	targets := otherPeers()

	if role == "leader" {
		needed := W - 1 // we already wrote locally so thats 1
		if W == 1 {
			// fire and forget - replicate in the background
			go replicateToNodes(req.Key, req.Value, version, targets, 0)
		} else {
			replicateToNodes(req.Key, req.Value, version, targets, needed)
		}
	} else {
		// leaderless mode: push to everyone, wait for all of them
		replicateToNodes(req.Key, req.Value, version, targets, len(targets))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{"version": version})
}

// sends the kv pair to each target one by one, 200ms gap between sends.
// once we hit requiredAcks we return and let a goroutine handle the rest.
func replicateToNodes(key, value string, version int, targets []string, requiredAcks int) {
	body, _ := json.Marshal(map[string]interface{}{
		"key": key, "value": value, "version": version,
	})

	acked := 0
	for i, target := range targets {
		resp, err := http.Post("http://"+target+"/internal/replicate", "application/json", bytes.NewReader(body))
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				acked++
			}
		} else {
			log.Printf("replicate to %s failed: %v", target, err)
		}
		time.Sleep(200 * time.Millisecond)

		// got enough? hand off the stragglers to a goroutine
		if acked >= requiredAcks && requiredAcks > 0 && i < len(targets)-1 {
			remaining := targets[i+1:]
			go func() {
				for _, t := range remaining {
					resp, err := http.Post("http://"+t+"/internal/replicate", "application/json", bytes.NewReader(body))
					if err == nil {
						resp.Body.Close()
					}
					time.Sleep(200 * time.Millisecond)
				}
			}()
			return
		}
	}
}

// GET /get/{key} - if R=1 just check local map, otherwise ask around
func handleGet(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]

	if R == 1 {
		storeMu.RLock()
		e, ok := store[key]
		storeMu.RUnlock()
		if !ok {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"key": key, "value": e.Value, "version": e.Version,
		})
		return
	}

	// need to check multiple nodes, pick whoever has the newest version
	best := gatherReads(key, R)
	if best.Version == 0 {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"key": key, "value": best.Value, "version": best.Version,
	})
}

// ask our peers + check locally, return whichever has the highest version.
// we need `needed` total responses (counting our own local read as one).
func gatherReads(key string, needed int) Entry {
	type result struct {
		entry Entry
		found bool
	}

	// check what we have locally first
	storeMu.RLock()
	localEntry, localOk := store[key]
	storeMu.RUnlock()

	targets := otherPeers()
	ch := make(chan result, len(targets))

	// fan out reads to everyone else at the same time
	for _, target := range targets {
		go func(t string) {
			resp, err := http.Get("http://" + t + "/internal/read/" + key)
			if err != nil || resp.StatusCode != http.StatusOK {
				if resp != nil {
					resp.Body.Close()
				}
				ch <- result{}
				return
			}
			var e Entry
			json.NewDecoder(resp.Body).Decode(&e)
			resp.Body.Close()
			ch <- result{entry: e, found: true}
		}(target)
	}

	// grab responses until we have enough, keep track of the highest version
	best := Entry{}
	if localOk {
		best = localEntry
	}
	collected := 1 // local counts
	for i := 0; i < len(targets) && collected < needed; i++ {
		r := <-ch
		if r.found && r.entry.Version > best.Version {
			best = r.entry
		}
		collected++
	}

	return best
}

// GET /local_read/{key} - sneaky test endpoint, just returns what THIS node has
func handleLocalRead(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	storeMu.RLock()
	e, ok := store[key]
	storeMu.RUnlock()
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"key": key, "value": e.Value, "version": e.Version,
	})
}

// POST /internal/replicate - other nodes call this to push updates to us
func handleInternalReplicate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key     string `json:"key"`
		Value   string `json:"value"`
		Version int    `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	time.Sleep(100 * time.Millisecond)

	storeMu.Lock()
	if existing, ok := store[req.Key]; !ok || req.Version > existing.Version {
		store[req.Key] = Entry{Value: req.Value, Version: req.Version}
	}
	storeMu.Unlock()

	w.WriteHeader(http.StatusOK)
}

// GET /internal/read/{key} - other nodes ask us what we have for a key
func handleInternalRead(w http.ResponseWriter, r *http.Request) {
	key := mux.Vars(r)["key"]
	time.Sleep(50 * time.Millisecond)

	storeMu.RLock()
	e, ok := store[key]
	storeMu.RUnlock()
	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(e)
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "ok")
}

func main() {
	initConfig()

	r := mux.NewRouter()
	r.HandleFunc("/set", handleSet).Methods("POST")
	r.HandleFunc("/get/{key}", handleGet).Methods("GET")
	r.HandleFunc("/local_read/{key}", handleLocalRead).Methods("GET")
	r.HandleFunc("/internal/replicate", handleInternalReplicate).Methods("POST")
	r.HandleFunc("/internal/read/{key}", handleInternalRead).Methods("GET")
	r.HandleFunc("/health", handleHealth).Methods("GET")

	log.Printf("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
