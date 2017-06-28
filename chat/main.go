package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"text/template"

	"github.com/TheGhostHuCodes/GPB/trace"
	"github.com/stretchr/gomniauth"
	"github.com/stretchr/gomniauth/providers/github"
	"github.com/stretchr/gomniauth/providers/google"
	"github.com/stretchr/objx"
)

type oauth2Provider struct {
	ClientId     string
	ClientSecret string
}

type oauth2Config struct {
	SecurityKey string
	Provider    map[string]oauth2Provider
}

// templ represents a single template
type templateHandler struct {
	once     sync.Once
	filename string
	templ    *template.Template
}

// ServeHTTP handles the HTTP request.
func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.once.Do(func() {
		t.templ = template.Must(template.ParseFiles(filepath.Join("templates",
			t.filename)))
	})
	data := map[string]interface{}{
		"Host": r.Host,
	}
	if authCookie, err := r.Cookie("auth"); err == nil {
		data["UserData"] = objx.MustFromBase64(authCookie.Value)
	}
	t.templ.Execute(w, data)
}

func main() {
	var addr = flag.String("addr", ":8080", "The addr of the applicaiton.")
	flag.Parse()

	// Setup gomniauth
	f, err := os.Open("oauth2_config.json")
	if err != nil {
		log.Fatalf("Could not open JSON config file.\nError: %s\n", err)
	}
	dec := json.NewDecoder(f)
	var c oauth2Config
	err = dec.Decode(&c)
	f.Close()
	if err != nil {
		log.Fatalf("Could not decode JSON config file.\nError: %s\n", err)
	}

	gomniauth.SetSecurityKey(c.SecurityKey)
	gomniauth.WithProviders(
		google.New(c.Provider["Google"].ClientId, c.Provider["Google"].ClientSecret,
			"http://localhost:8080/auth/callback/google"),
		github.New(c.Provider["Github"].ClientId, c.Provider["Github"].ClientSecret,
			"http://localhost:8080/auth/callback/github"),
	)

	r := newRoom()
	r.tracer = trace.New(os.Stdout)
	http.Handle("/chat", MustAuth(&templateHandler{filename: "chat.html"}))
	http.Handle("/login", &templateHandler{filename: "login.html"})
	http.HandleFunc("/auth/", loginHandler)
	http.Handle("/room", r)
	// get the room going
	go r.run()
	// Start the web server
	log.Println("Starting web server on", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
