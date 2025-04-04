// This file is part of Gopher2600.
//
// Gopher2600 is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Gopher2600 is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with Gopher2600.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"log"
	"net/http"
	"strings"
)

type handler struct {
	fileHandler http.Handler
}

func Handler() *handler {
	hnd := handler{
		fileHandler: http.FileServer(http.Dir("www")),
	}
	return &hnd
}

func (hnd *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("%s %s", r.Method, r.RequestURI)
	if strings.HasSuffix(r.RequestURI, ".wasm") {
		w.Header().Set("Content-Type", "application/wasm")
	}
	hnd.fileHandler.ServeHTTP(w, r)
}

func main() {
	log.Println("test server listening on localhost:7800")
	err := http.ListenAndServe(":7800", Handler())
	if err != nil {
		log.Fatalln(err.Error())
	}
}
