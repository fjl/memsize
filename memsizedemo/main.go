package main

import (
	"net/http"

	"github.com/fjl/memsize/memsizeui"
)

func main() {
	data := make([]byte, 200)

	h := new(memsizeui.Handler)
	s := &http.Server{Addr: ":8080", Handler: h}
	h.Add("", &data)
	s.ListenAndServe()
}
