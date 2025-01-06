package main

import (
	"flag"
	"fmt"
)

func main() {
	port := flag.Int("port", 8080, "Port on which the caching proxy server will run")
	origin := flag.String("origin", "", "Origin server to which requests will be forwarded")
	flag.Parse()

	fmt.Println(*port, *origin)
}
