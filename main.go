package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"text/template"
)

var (
	host, port, layout, reqPath, actPath string
	archive, poems                       []string
)

// postAttr contains template tags
type postAttr struct {
	Title, Date, Banner, Content string
	IsArchive, IsPoems           bool
}

// runHTTP runs the server on the given listen address
// and sets the routing handlers
func runHTTP(listenAddr string) error {
	s := http.Server{
		Addr:    listenAddr,
		Handler: newRouter(), // own instance of servemux
	}

	fmt.Printf("Starting HTTP listener at %s\n", listenAddr)
	return s.ListenAndServe()
}

// triggerLoader loads the config from
// loader.txt file in the project root directory
func triggerLoader() {
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
			case "[config]":
				var cfg []string
				cfg = append(cfg, configLines[i+1:3]...)
				host = strings.Split(cfg[0], "=")[1]
				port = strings.Split(cfg[1], "=")[1]
			case "[layout]":
				layout = configLines[i+1]
			case "[archive]":
				for v := i + 1; v < len(configLines); v++ {
					if strings.HasPrefix(configLines[v], "[") || configLines[v] == "" {
						break
					}
					archive = append(archive, configLines[v])
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
}

// postHandler serves dynamic .txt posts and currently
// using for /archive/post1 and poems/poem1
func postHandler(w http.ResponseWriter, r *http.Request) {
	var pT postAttr
	reqPath = r.URL.Path
	if reqPath == "" || strings.HasSuffix(reqPath, "/") || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	if strings.Split(reqPath, "/")[1] == "archive" {
		pT.IsArchive = true
	}

	if strings.Split(reqPath, "/")[1] == "poems" {
		pT.IsPoems = true
	}

	actPath = fmt.Sprintf(".%s.txt", reqPath)

	content, err := os.ReadFile(actPath)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, http.StatusText(500), 500)
		return
	}

	sFront := strings.Split(string(content), "\n")[:3]

	sTitle := sFront[0]
	sDate := sFront[1]

	sContent := strings.Split(string(content), "\n")[3:]

	var banner string
	banner = strings.Join(sContent[:2], "<br>")
	if !strings.HasPrefix(banner, "<img") {
		banner = ""
	}
	article := strings.Join(sContent[2:], "")

	pT.Title = sTitle
	pT.Date = sDate
	pT.Content = article
	pT.Banner = banner

	tmpl, err := template.ParseFiles(layout)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, http.StatusText(500), 500)
		return
	}

	err = tmpl.ExecuteTemplate(w, "layout", pT)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, http.StatusText(500), 500)
	}
}

// htmlFileHandler serves HTML files directly without
// any manipulation and currently using index.html pages
// in the project root and categories.
func htmlFileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	switch r.URL.Path {
	case "/":
		http.ServeFile(w, r, "index.html")
		return
	// case "/license":
	// 	http.ServeFile(w, r, "license.html")
	// 	return
	case "/archive":
		http.ServeFile(w, r, "./archive/index.html")
		return
	case "/poems":
		http.ServeFile(w, r, "./poems/index.html")
		return
	case "/art":
		http.ServeFile(w, r, "./art/index.html")
		return
	default:
		http.NotFound(w, r)
		return
	}
}

// newRouter is a registry of routers
func newRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", htmlFileHandler)
	mux.HandleFunc("/license", htmlFileHandler)
	mux.HandleFunc("/art", htmlFileHandler)
	mux.HandleFunc("/archive", htmlFileHandler)
	mux.HandleFunc("/archive/", postHandler)
	mux.HandleFunc("/poems", htmlFileHandler)
	mux.HandleFunc("/poems/", postHandler)

	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets"))))

	return mux
}

func main() {
	triggerLoader()
	addr := net.JoinHostPort(host, port)
	if err := runHTTP(addr); err != nil {
		log.Fatal(err)
	}
}
