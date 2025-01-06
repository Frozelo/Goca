package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
)

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
		w.Write([]byte(r.Method))
	})

	addr := fmt.Sprintf(":%d", *port)

	log.Printf("Starting caching proxy on port %d, forwarding to %s", *port, originUrl)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}

}
