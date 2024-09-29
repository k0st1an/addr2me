package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
)

var (
	addr      = flag.String("port", ":7007", "port to listen on")
	logPrefix = flag.String("log-prefix", "[addr2.me] ", "log prefix")
)

func init() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.SetPrefix(*logPrefix)
}

func getClientIP(w http.ResponseWriter, r *http.Request) {
	client_ip := r.Header.Get("X-Real-Ip")

	if client_ip == "" {
		client_ip = r.Header.Get("X-Forwarded-For")
	}

	if client_ip == "" {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Println(err)
		}

		client_ip = host
	}

	fmt.Fprintf(w, "%s\n", client_ip)
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	flag.Parse()

	http.HandleFunc("/", getClientIP)

	log.Printf("Listening on http://%s", *addr)
	err := http.ListenAndServe(*addr, nil)
	if err != nil {
		log.Fatalln(err)
	}
}
