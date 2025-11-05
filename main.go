package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	mux := http.NewServeMux()

	// Home route returns static HTML
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `
			<!doctype html>
			<html>
				<head><meta charset="utf-8"><title>Go CRM API</title></head>
				<body>
					<h1>Go CRM API</h1>
					<p>Available endpoints:</p>
					<ul>
						<li>GET /customers</li>
						<li>GET /customers/{id}</li>
						<li>POST /customers</li>
						<li>PUT /customers/{id}</li>
						<li>DELETE /customers/{id}</li>
					</ul>
				</body>
			</html>
		`)
	})

	log.Println("Server started on port 3000")
	log.Fatal(http.ListenAndServe(":3000", mux))
}