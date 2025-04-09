package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"

	"github.com/google/uuid"
	_ "github.com/jimmicro/version"
)

var address = "127.0.0.1:8080"

func main() {
	flag.StringVar(&address, "address", "127.0.0.1:8080", "listen address")
	flag.Parse()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		uuid := uuid.New().String()
		logger := log.New(os.Stdout, uuid+" ", log.Ldate|log.Ltime|log.Lmsgprefix)
		rawURL := r.URL.Query().Get("url")
		if rawURL == "" {
			http.Error(w, "url is required", http.StatusBadRequest)
			return
		}
		logger.Printf("target: %s", rawURL)
		req, err := http.NewRequest(r.Method, rawURL, r.Body)
		if err != nil {
			http.Error(w, "Failed to create request: "+err.Error(), http.StatusInternalServerError)
			return
		}

		for key, values := range r.Header {
			for _, v := range values {
				req.Header.Add(key, v)
			}
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, "Upstream fetch failed: "+err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()

		for key, values := range resp.Header {
			for _, v := range values {
				w.Header().Add(key, v)
			}
		}
		parsedURL, err := url.Parse(rawURL)
		if err != nil {
			http.Error(w, "Invalid URL: "+err.Error(), http.StatusBadRequest)
			return
		}
		filename := path.Base(parsedURL.Path)
		w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
		w.WriteHeader(resp.StatusCode)

		_, err = io.Copy(w, resp.Body)
		if err != nil {
			http.Error(w, "Failed to copy response body: "+err.Error(), http.StatusInternalServerError)
			return
		}
		logger.Printf("size: %d", resp.ContentLength)
	})

	log.Printf("Listening on %s", address)
	log.Fatal(http.ListenAndServe(address, nil))
}
