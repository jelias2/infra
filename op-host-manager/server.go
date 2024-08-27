package ophostmanager

import (
	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/gorilla/mux"
	"net/http"
)

type Server struct {
	router *mux.Router
	port   int
	ohm    *OPHostManager
}

func NewServer() (*Server, error) {

	r := mux.NewRouter()
	port := 8080
	r.HandleFunc("/", HandleHealthz)
	http.Handle("/", r)

	ohm, err := NewOPHostMananger()
	if err != nil {
		log.Error("Error creating ophostmananger",
			"err", err,
		)
		return nil, err
	}

	return &Server{
		router: r,
		port:   port,
		ohm:    ohm,
	}, nil

}

func (s *Server) Start() {
	go func() {
		if err := (http.ListenAndServe(fmt.Sprintf(":%d", s.port), nil)); err != nil {
			log.Error("error starting metrics server", "err", err)
		}
	}()
}

func HandleHealthz(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("OK"))
}
