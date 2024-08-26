package server

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/mux"
	"net/http"
	"time"
)

type Server struct {
	router *mux.Router
	port   int
}

func NewServer() Server {

	r := mux.NewRouter()
	port := 8080
	r.HandleFunc("/", HandleHealthz)
	http.Handle("/", r)

	return Server{
		router: r,
		port:   port,
	}

}

func (s *Server) Start() {
	srv := &http.Server{
		Handler: s.router,
		Addr:    "127.0.0.1:8000",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Crit("Error with server")
	}
}

func HandleHealthz(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("OK"))
}
