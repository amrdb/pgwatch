package webui

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type WebUIServer struct {
	// l        log.Logger
	http.Server
	PgWatchVersion  string
	PostgresVersion string
	GrafanaVersion  string
}

//go:embed build
var UI embed.FS
var uiFS fs.FS

func Init(addr string) *WebUIServer {
	mux := http.NewServeMux()
	var err error
	uiFS, err = fs.Sub(UI, "build")
	if err != nil {
		log.Fatal("failed to get ui fs", err)
	}

	s := &WebUIServer{
		// nil,
		// logger,
		http.Server{
			Addr:           addr,
			ReadTimeout:    10 * time.Second,
			WriteTimeout:   10 * time.Second,
			MaxHeaderBytes: 1 << 20,
			Handler:        mux,
		},
		"3.0.0", "14.4", "8.7.0",
	}

	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api", s.handleApi)
	mux.HandleFunc("/", s.handleStatic)

	if 8080 != 0 {
		go func() { panic(s.ListenAndServe()) }()
	}
	return s
}

func (Server *WebUIServer) handleStatic(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	path := r.URL.Path
	if path == "/" { // Add other paths that you route on the UI-side here
		path = "index.html"
	}
	path = strings.TrimPrefix(path, "/")

	file, err := uiFS.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("file", path, "not found:", err)
			http.NotFound(w, r)
			return
		}
		log.Println("file", path, "cannot be read:", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	contentType := mime.TypeByExtension(filepath.Ext(path))
	w.Header().Set("Content-Type", contentType)
	if strings.HasPrefix(path, "static/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000")
	}
	stat, err := file.Stat()
	if err == nil && stat.Size() > 0 {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", stat.Size()))
	}

	n, _ := io.Copy(w, file)
	log.Println("file", path, "copied", n, "bytes")
}

func (Server *WebUIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("versions").Parse(`{{define "title"}}Versions{{end}}
<html>
<body>
<ul>
    <li>pgwatch3 {{ .PgWatchVersion }}</li>
    <li>Grafana {{ .GrafanaVersion }}</li>
    <li>Postgres {{ .PostgresVersion }}</li>
</ul>
</body>
</html>`)

	if err != nil {
		log.Print(err.Error())
		http.Error(w, http.StatusText(500), 500)
		return
	}

	err = tmpl.ExecuteTemplate(w, "versions", Server)
	if err != nil {
		log.Print(err.Error())
		http.Error(w, http.StatusText(500), 500)
	}
}

func (Server *WebUIServer) handleApi(w http.ResponseWriter, r *http.Request) {
	// TODO
}
