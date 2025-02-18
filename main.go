package main

import (
	"fmt"
	// htmltemplate "html/template"
	// "io"
	"net/http"
	// "os"
	// "os/exec"
	// "regexp"
	// texttemplate "text/template"
)

func handleRoot(
	w http.ResponseWriter,
	r *http.Request,
) {
	fmt.Fprintf(w, "Admin Page")
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)

	fmt.Println("Server started at http://localhost:8080")
	http.ListenAndServe(":8080", mux)
}
