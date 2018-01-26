package main

import (
	"net/http"

	"github.com/fjl/memsize/memsizeui"
)

func main() {
	byteslice := make([]byte, 200)
	intslice := make([]int, 100)

	h := new(memsizeui.Handler)
	s := &http.Server{Addr: ":8080", Handler: h}
	h.Add("byteslice", &byteslice)
	h.Add("intslice", &intslice)
	s.ListenAndServe()
}
