package server

import (
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/mux"
	"net/http"
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
	go func() {
		if err := (http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)); err != nil {
			log.Crit("Error running server, exiting", err)
		}
	}()
}

func HandleHealthz(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("OK"))
}
