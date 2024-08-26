package server

import (
	"github.com/gorilla/mux"
	"net/http"
)

func HandleHealthz(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("OK"))
}
