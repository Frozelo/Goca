package main

import (
	"context"
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
	ExpiresAt  time.Time
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

func (ic *InMemmoryCache) CleanExpired() {
	now := time.Now()
	ic.mu.Lock()
	defer ic.mu.Unlock()

	for k, item := range ic.data {
		itemExpires := item.ExpiresAt
		if now.After(itemExpires) {
			log.Println("Deleting expired cache", k)
			delete(ic.data, k)
		}
	}
}

var cache = NewCache()

func main() {
	port := flag.Int("port", 8080, "Port on which the caching proxy server will run")
	origin := flag.String("origin", "", "Origin server to which requests will be forwarded")
	ttl := flag.Duration("cache-ttl", 30*time.Second, "Cache TTL duration")
	cleanupInterval := flag.Duration("cleanup-interval", 10*time.Second, "Cleanup interval for removing expired items from cache")
	flag.Parse()

	if *origin == "" {
		log.Fatal("You must provide origin url")
	}

	originUrl, err := url.Parse(*origin)
	if err != nil {
		log.Fatalf("Failed to parse origin: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		ticker := time.NewTicker(*cleanupInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				log.Println("Cleaning expired items inside the cache...")
				cache.CleanExpired()
			case <-ctx.Done():
				log.Println("Stopping cache cleanup goroutine")
				return
			}
		}
	}()

	http.HandleFunc("/projects", func(w http.ResponseWriter, r *http.Request) {
		handleRequest(w, r, originUrl, *ttl)
	})

	addr := fmt.Sprintf(":%d", *port)

	log.Printf("Starting caching proxy on port %d, forwarding to %s", *port, originUrl)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func handleRequest(w http.ResponseWriter, r *http.Request, originUrl *url.URL, ttl time.Duration) {
	cacheKey := r.Method + ":" + originUrl.String() + r.URL.RequestURI()

	if item, exists := cache.Get(cacheKey); exists {
		log.Println("Found in it cache")
		w.Header().Set("X-Cache", "HIT")
		w.Write(item.Body)
		return
	}
	log.Println("Not found in cache. Providing request to origin url and save it in cache")
	forwardUrl := *originUrl
	endpont := "/api/v1/projects/"

	req, err := http.NewRequestWithContext(r.Context(), r.Method, forwardUrl.String()+endpont, r.Body)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to create request to origin", http.StatusInternalServerError)
		return
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		log.Println(err)
		http.Error(w, "Failed to get response from origin", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Failed to get bodyBytes from body response", http.StatusInternalServerError)
		return
	}

	now := time.Now()
	item := CacheItem{
		Body:       bodyBytes,
		Header:     resp.Header,
		StatusCode: resp.StatusCode,
		CachedAt:   time.Now(),
		ExpiresAt:  now.Add(ttl),
	}

	log.Println("Saving in cache")
	cache.Set(cacheKey, &item)

	w.Header().Set("X-Cache", "MISS")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(bodyBytes)
}
