package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"crypto/rand"
)

type Customer struct {
	ID string 
	Name string 
	Role string
	Email string 
	Phone int   
	Contacted bool  
}


var (
	dbMu sync.RWMutex
	customers = make(map[string]Customer)
)

func seed() {
	c1 := Customer{ID: generateCustomerId(), Name: "Jordan Holland", Role: "Software Engineer", Email: "hollandjb@blah.com", Phone: 5551209, Contacted: true}
	c2 := Customer{ID: generateCustomerId(), Name: "Matthew Santiago", Role: "Support Engineer", Email: "test@example.com", Phone: 8289028, Contacted: false}
	c3 := Customer{ID: generateCustomerId(), Name: "Abigail Spanberger", Role: "Governor", Email: "abigail@example.com", Phone: 2329752, Contacted: true}

	dbMu.Lock()
	customers[c1.ID] = c1
	customers[c2.ID] = c2
	customers[c3.ID] = c3
	dbMu.Unlock()
}


// generateCustomerId returns a 32-hex char id (simple, unique enough for this app).
func generateCustomerId() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// fallback; use address
		return fmt.Sprintf("%p", &b)
	}

	const hexdigits = "0123456789abcdef"
	out := make([]byte, 32)
	for i, by := range b {
		out[i*2] = hexdigits[by>>4]
		out[i*2+1] = hexdigits[by&0x0f]
	}
	return string(out)
}

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