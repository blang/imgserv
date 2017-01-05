package main

import (
	"container/ring"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	path "path/filepath"
	"sync"
	"time"
)

var fileBuffer = NewFileBuffer(1024)

type FileBuffer struct {
	buf *ring.Ring
	cap int
	sync.Mutex
}

func NewFileBuffer(size int) *FileBuffer {
	return &FileBuffer{
		cap: size,
		buf: ring.New(size),
	}
}

func (b *FileBuffer) Append(s string) {
	b.Lock()
	vals := b.slice()
	if len(vals) == b.cap {
		os.Remove(path.Join(*filepath, vals[0]))
	}
	b.buf.Value = s
	b.buf = b.buf.Next()
	b.Unlock()
}

func (b *FileBuffer) slice() []string {
	var vals []string
	b.buf.Do(func(f interface{}) {
		if f != nil {
			vals = append(vals, f.(string))
		}
	})
	return vals
}
func (b *FileBuffer) Slice() []string {
	b.Lock()
	vals := b.slice()
	b.Unlock()
	return vals
}

func upload(filepath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			if paused {
				http.Error(w, "", http.StatusServiceUnavailable)
				return
			}
			r.ParseMultipartForm(32 << 20)
			file, _, err := r.FormFile("file")
			if err != nil {
				log.Printf("Error uploading: %s", err)
				http.Error(w, "Error", 500)
				return
			}
			defer file.Close()
			name := fmt.Sprintf("%s.jpg", time.Now().Format("20060102-15:04:05.000"))
			filename := path.Join(filepath, name)
			f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
			if err != nil {
				log.Printf("Error uploading: %s", err)
				http.Error(w, "Error", 500)
				return
			}
			defer f.Close()
			_, err = io.Copy(f, file)
			if err != nil {
				log.Printf("Error uploading: %s", err)
				http.Error(w, "Error", 500)
				return
			}
			fileBuffer.Append(name)
			fmt.Fprintln(w, "Success!")

		} else {
			http.Error(w, "Not found", 404)
		}
	})
}

const tmp = `
<html><head><meta http-equiv="refresh" content="5; URL=/last"><title>Img</title></head>
<body>
<img src="img.jpg" style="height: 90%">
</body></html>
`

func img(filename string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vals := fileBuffer.Slice()
		if len(vals) > 0 {
			http.ServeFile(w, r, path.Join(filename, vals[len(vals)-1]))
		}
	})
}

func last(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, tmp)
}
func index(filepath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "<html><body><h1>Bilder</h1>\n")
		for _, v := range fileBuffer.Slice() {
			fmt.Fprintf(w, `<a href="%s">%s</a><br />`+"\n", path.Join("/imgs", v), v)
		}
		fmt.Fprintf(w, "</body></html>\n")
	})
}

func auth(user, pass string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		inuser, inpass, ok := r.BasicAuth()
		if !ok || inuser != user || inpass != pass {
			w.Header().Set("WWW-Authenticate", "Basic realm=\"RealmName\"")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func upauth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.FormValue("token") != token {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

var (
	username    = flag.String("username", "", "Username for basic auth")
	password    = flag.String("password", "", "Password for basic auth")
	uploadtoken = flag.String("uploadtoken", "", "Token for upload")
	listen      = flag.String("listen", ":12345", "Listen")
	filepath    = flag.String("path", "/tmp/", "Filepath to image")
)

var paused = false

func pause(w http.ResponseWriter, r *http.Request) {
	if paused {
		paused = false
		fmt.Fprintln(w, "Unpaused")
	} else {
		paused = true
		fmt.Fprintln(w, "Paused")
	}
}

func main() {
	flag.Parse()
	http.Handle("/imgs/", auth(*username, *password, http.StripPrefix("/imgs/", http.FileServer(http.Dir(*filepath)))))
	http.Handle("/img.jpg", auth(*username, *password, img(*filepath)))
	http.Handle("/upload", upauth(*uploadtoken, upload(*filepath)))
	http.Handle("/pause", auth(*username, *password, http.HandlerFunc(pause)))
	http.Handle("/last", auth(*username, *password, http.HandlerFunc(last)))
	http.Handle("/", auth(*username, *password, index(*filepath)))
	log.Fatal(http.ListenAndServe(*listen, nil))
}
