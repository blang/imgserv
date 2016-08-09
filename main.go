package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

func upload(filename string) http.Handler {
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
			f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0666)
			if err != nil {
				log.Printf("Error uploading: %s", err)
				http.Error(w, "Error", 500)
				return
			}
			defer f.Close()
			io.Copy(f, file)
			fmt.Fprintln(w, "Success!")
		} else {
			http.Error(w, "Not found", 404)
		}
	})
}

const tmp = `
<html><head><meta http-equiv="refresh" content="5; URL=/"><title>Img</title></head>
<body>
<img src="img.jpg" style="height: 90%">
</body></html>
`

func img(filename string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, filename)
	})
}
func index(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, tmp)
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
		log.Println(r.RequestURI)
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
	filepath    = flag.String("path", "/tmp/img.jpg", "Filepath to image")
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
	http.Handle("/", auth(*username, *password, http.HandlerFunc(index)))
	http.Handle("/img.jpg", auth(*username, *password, img(*filepath)))
	http.Handle("/upload", upauth(*uploadtoken, upload(*filepath)))
	http.Handle("/pause", auth(*username, *password, http.HandlerFunc(pause)))
	log.Fatal(http.ListenAndServe(*listen, nil))
	// upload logic
}
