package main

import (
	"bufio"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/yuin/goldmark"
)

const poemHTMLTop = `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><meta http-equiv="X-UA-Compatible" content="IE=edge"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>mysticmode - All my public musing</title><link rel="stylesheet" href="/assets/css/style.css"></head><body><nav><a href="/">home</a><a href="/archive">archive</a><a href="https://github.com/mysticmode">code</a><a href="/poem" class="active">poems</a></nav><main>`
const archiveHTMLTop = `<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8"><meta http-equiv="X-UA-Compatible" content="IE=edge"><meta name="viewport" content="width=device-width, initial-scale=1.0"><title>mysticmode - All my public musing</title><link rel="stylesheet" href="/assets/css/style.css"></head><body><nav><a href="/">home</a><a href="/archive" class="active">archive</a><a href="https://github.com/mysticmode">code</a><a href="/poem">poems</a></nav><main>`
const htmlBottom = `</main></body></html>`

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

type IndexAttribute struct {
	Year string
	Post map[string]string // map[link]title
}

func genIndex(dirName string) {
	var postIndex []IndexAttribute
	var iA IndexAttribute

	files, err := ioutil.ReadDir(dirName)
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < len(files); i++ {
		fN := files[i].Name()

		// filename format => YYYY-MM-DD_post-title.html
		fY := strings.Split(strings.Split(fN, "_")[0], "-")[0]
		fO, err := os.OpenFile(filepath.Join(dirName, fN), os.O_RDONLY, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
		defer func() {
			if err := fO.Close(); err != nil {
				log.Fatal(err)
			}
		}()

		var fT string
		scanner := bufio.NewScanner(fO)
		if scanner.Scan() {
			fT = scanner.Text()
		}

		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}

		if fY == iA.Year {
			iA.Post[fN] = fT
		} else {
			if iA.Post != nil {
				postIndex = append(postIndex, iA)
			}
			iA = IndexAttribute{}
			iA.Post = make(map[string]string)
			iA.Year = fY
			iA.Post[fN] = fT
		}
	}

	indexLayout, err := os.OpenFile(filepath.Join(dirName, "index.html"), os.O_RDONLY, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := indexLayout.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	indexFile, err := os.OpenFile(filepath.Join("docs", dirName, "index.html"), os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := indexFile.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	tmpl, err := template.New("post-index").ParseFiles(indexLayout.Name())
	if err != nil {
		log.Fatal(err)
	}

	err = tmpl.Execute(indexFile, postIndex)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(postIndex)
}

type formValue struct {
	title, year, month, day, content, postType string
}

func submitPost(w http.ResponseWriter, r *http.Request) {
	fV := formValue{
		title:    r.FormValue("title"),
		year:     r.FormValue("year"),
		month:    r.FormValue("month"),
		day:      r.FormValue("day"),
		content:  r.FormValue("content"),
		postType: r.FormValue("type"),
	}

	tS := strings.ReplaceAll(fV.title, " ", "-")

	fileName := fmt.Sprintf("%s-%s-%s_%s", fV.year, fV.month, fV.day, tS)

	var fO *os.File
	var err error

	if fV.postType == "poem" {
		fO, err = os.Create(filepath.Join("poem", fileName+".md"))
	} else {
		fO, err = os.Create(filepath.Join("archive", fileName+".md"))
	}
	defer func() {
		if err := fO.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	if err != nil {
		log.Fatal(err)
	}

	bW := bufio.NewWriter(fO)

	// write post title as a first line
	bW.Write([]byte(fmt.Sprintf("%s\n", fV.title)))

	// write post date as a second line
	bW.Write([]byte(fmt.Sprintf("%s-%s-%s\n", fV.year, fV.month, fV.day)))

	bW.Write([]byte(fV.content))

	bW.Flush()

	var fAct *os.File
	var htmlTop string

	if fV.postType == "poem" {
		htmlTop = poemHTMLTop
		fAct, err = os.Create(filepath.Join("docs/poem", fileName+".html"))
	} else {
		htmlTop = archiveHTMLTop
		fAct, err = os.Create(filepath.Join("docs/archive", fileName+".html"))
	}
	defer func() {
		if err := fAct.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	if err != nil {
		log.Fatal(err)
	}

	bWAct := bufio.NewWriter(fAct)

	bWAct.Write([]byte(htmlTop))
	bWAct.Write([]byte(fmt.Sprintf("<h3>%s</h3>", fV.title)))
	bWAct.Write([]byte(fmt.Sprintf("<span>%s-%s-%s</span>", fV.year, fV.month, fV.day)))
	bWAct.Write([]byte("<div>"))

	if err := goldmark.Convert([]byte(fV.content), bWAct); err != nil {
		log.Fatal(err)
	}

	bWAct.Write([]byte("</div>"))
	bWAct.Write([]byte(htmlBottom))

	bWAct.Flush()

	// go genIndex(fV.postType)

	w.Write([]byte("Posted successfully!"))
}

// newRouter is a registry of routers
func newRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			submitPost(w, r)
			return
		}

		http.ServeFile(w, r, "write.html")
	})

	mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.Dir("./docs/assets"))))

	return mux
}

func main() {
	addr := net.JoinHostPort("localhost", "4000")
	if err := runHTTP(addr); err != nil {
		log.Fatal(err)
	}
}
