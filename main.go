package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
)

// host vars
var (
	host = flag.String("host", "", "host http address to listen on")
	port = flag.String("port", "8000", "port number for http listener")
)

// blog
var blog = []string{"index.html", "2021-11-28_hello-from-mysticmode.html", "2020-11-10_about-annamalai-swami.html"}

// poems
var poems = []string{"index.html", "2022-01-25_dont-just-get-going.html", "2021-08-12_life-is-preciously-short.html", "2014-09-09_solitude.html"}

func runHTTP(listenAddr string) error {
	s := http.Server{
		Addr:    listenAddr,
		Handler: newRouter(), // own instance of servemux
	}

	fmt.Printf("Starting HTTP listener at %s\n", listenAddr)
	return s.ListenAndServe()
}

func newRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}

		http.ServeFile(w, r, "index.html")
	})

	mux.HandleFunc("/art", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/art" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}

		http.ServeFile(w, r, "./art/index.html")
	})

	for _, b := range blog {
		var rpth, apth string

		ripHtml := strings.Split(b, ".html")[0]
		if ripHtml == "index" {
			rpth = "/blog"
		} else {
			rpth = fmt.Sprintf("/blog/%s", ripHtml)
		}

		apth = fmt.Sprintf("./blog/%s", b)

		mux.HandleFunc(rpth, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.NotFound(w, r)
				return
			}

			http.ServeFile(w, r, apth)
		})
	}

	for _, p := range poems {
		var rpth, apth string

		ripHtml := strings.Split(p, ".html")[0]
		if ripHtml == "index" {
			rpth = "/poems"
		} else {
			rpth = fmt.Sprintf("/poems/%s", ripHtml)
		}

		apth = fmt.Sprintf("./poems/%s", p)

		mux.HandleFunc(rpth, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.NotFound(w, r)
				return
			}

			http.ServeFile(w, r, apth)
		})
	}

	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))

	return mux
}

func main() {
	flag.Parse()
	addr := net.JoinHostPort(*host, *port)
	if err := runHTTP(addr); err != nil {
		log.Fatal(err)
	}
}
