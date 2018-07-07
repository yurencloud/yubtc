package controller

import (
	"net/http"
)

func Index(w http.ResponseWriter, r *http.Request) {
	w.Write( []byte("Hello World"))
}

func Home(w http.ResponseWriter, r *http.Request) {
	w.Write( []byte("Pass Auth to Home"))
}
