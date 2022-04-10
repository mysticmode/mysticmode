package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"text/template"
)

// host vars
var (
	host = flag.String("host", "", "host http address to listen on")
	port = flag.String("port", "4000", "port number for http listener")
)

// layout path
var (
	layoutPath = "layout.html"
)

// blog
var blog []string

// poems
var poems []string

type blogStore struct {
	Title   string
	Date    string
	Content string
}

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

	f, err := os.Open("loader.txt")
	if err != nil {
		log.Fatal("failed to open config file")
	}

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	var configLines []string

	for scanner.Scan() {
		configLines = append(configLines, scanner.Text())
	}

	f.Close()

	for i, line := range configLines {
		if strings.HasPrefix(line, "[") {
			switch line {
			case "[blog]":
				for v := i + 1; v < len(configLines); v++ {
					if strings.HasPrefix(configLines[v], "[") || configLines[v] == "" {
						break
					}
					blog = append(blog, configLines[v])
				}
			case "[poems]":
				for v := i + 1; v < len(configLines); v++ {
					if strings.HasPrefix(configLines[v], "[") || configLines[v] == "" {
						break
					}
					poems = append(poems, configLines[v])
				}
			}
		}
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}

		http.ServeFile(w, r, "index.html")
	})

	mux.HandleFunc("/license", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/license" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}

		http.ServeFile(w, r, "license.html")
	})

	mux.HandleFunc("/art", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/art" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}

		http.ServeFile(w, r, "./art/index.html")
	})

	mux.HandleFunc("/blog", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/blog" || r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}

		http.ServeFile(w, r, "./blog/index.html")
	})

	for _, b := range blog {
		var reqPath, actPath string

		ripExt := strings.Split(b, ".txt")[0]
		reqPath = fmt.Sprintf("/blog/%s", ripExt)

		actPath = fmt.Sprintf("./blog/%s", b)

		mux.HandleFunc(reqPath, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				http.NotFound(w, r)
				return
			}

			info, err := os.Stat(layoutPath)
			if err != nil {
				if os.IsNotExist(err) {
					http.NotFound(w, r)
					return
				}
			}

			if info.IsDir() {
				http.NotFound(w, r)
				return
			}

			content, err := os.ReadFile(actPath)
			if err != nil {
				log.Println(err.Error())
				http.Error(w, http.StatusText(500), 500)
				return
			}

			sFront := strings.Split(string(content), "\n")[0:3]

			sTitle := strings.Split(sFront[0], "->")[1]
			sDate := strings.Split(sFront[1], "->")[1]

			sContent := strings.Split(string(content), "\n")[3:]
			article := strings.Join(sContent, "<br>")

			bT := blogStore{
				Title:   sTitle,
				Date:    sDate,
				Content: article,
			}

			tmpl, err := template.ParseFiles(layoutPath)
			if err != nil {
				log.Println(err.Error())
				http.Error(w, http.StatusText(500), 500)
				return
			}

			err = tmpl.ExecuteTemplate(w, "layout", bT)
			if err != nil {
				log.Println(err.Error())
				http.Error(w, http.StatusText(500), 500)
			}
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
