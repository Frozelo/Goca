package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type CacheItem struct {
	Body       []byte
	Header     http.Header
	StatusCode int
	CachedAt   time.Time
}

type InMemmoryCache struct {
	data map[string]*CacheItem
	mu   sync.RWMutex
}

func NewCache() *InMemmoryCache {
	return &InMemmoryCache{
		data: make(map[string]*CacheItem),
	}
}

func (ic *InMemmoryCache) Get(key string) (*CacheItem, bool) {
	ic.mu.RLock()
	defer ic.mu.RUnlock()

	item, exists := ic.data[key]

	return item, exists
}

func (ic *InMemmoryCache) Set(key string, item *CacheItem) {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	ic.data[key] = item

}

var cache = NewCache()

func main() {
	port := flag.Int("port", 8080, "Port on which the caching proxy server will run")
	origin := flag.String("origin", "", "Origin server to which requests will be forwarded")
	flag.Parse()

	if *origin == "" {
		log.Fatal("You must provide origin url")
	}

	originUrl, err := url.Parse(*origin)
	if err != nil {
		log.Fatal("Failed to parse origin: %w", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handleRequest(w, r, originUrl)
	})

	addr := fmt.Sprintf(":%d", *port)

	log.Printf("Starting caching proxy on port %d, forwarding to %s", *port, originUrl)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request, originUrl *url.URL) {
	cacheKey := r.Method + ":" + originUrl.String()

	if item, exists := cache.Get(cacheKey); exists {
		log.Println("Found in it cache")
		w.Header().Set("X-Cache", "HIT")
		w.Write(item.Body)
		return
	}
	log.Println("Not found in cache. Providing request to origin url and save it in cache")
	forwardUrl := *originUrl

	req, err := http.NewRequest(r.Method, forwardUrl.String(), r.Body)
	if err != nil {
		http.Error(w, "Failed to create request to origin", http.StatusInternalServerError)
		return
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to get response from origin", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to get bodyBytes from body response", http.StatusInternalServerError)
		return
	}

	item := CacheItem{
		Body:       bodyBytes,
		Header:     resp.Header,
		StatusCode: resp.StatusCode,
		CachedAt:   time.Now(),
	}

	log.Println("Saving in cache")
	cache.Set(cacheKey, &item)

	w.Header().Set("X-Cache", "MISS")
	w.WriteHeader(http.StatusOK)
	w.Write(bodyBytes)
}
