package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"sync"
	"encoding/json"
	"strings"
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

// gets the ID in the /customers/{id} path
func parseCustomerId(path string) string {
	trimmed := strings.TrimSuffix(path, "/")
	const prefix = "/customers/"
	if strings.HasPrefix(trimmed, prefix) {
		return trimmed[len(prefix):]
	}
	return ""
}

// converts a map to a slice for JSON response
func toSlice(m map[string]Customer) []Customer {
	out := make([]Customer, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}


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



func getCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	id := parseCustomerId(r.URL.Path)
	if id == "" {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Customer not found"})
		return
	}

	dbMu.RLock()
	c, ok := customers[id]
	dbMu.RUnlock()

	if !ok {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(nil)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(c)
}


func getCustomers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	dbMu.RLock()
	list := toSlice(customers)
	dbMu.RUnlock()

	json.NewEncoder(w).Encode(list)
}


// to be implemented
 func addCustomer(w http.ResponseWriter, r *http.Request) {

 }

 func updateCustomer(w http.ResponseWriter, r *http.Request) {

 }

 func deleteCustomer(w http.ResponseWriter, r *http.Request) {

 }

func main() {
	seed()

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

	
	mux.HandleFunc("/customers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getCustomers(w, r)
		case http.MethodPost:
			addCustomer(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	// routes with an ID
	mux.HandleFunc("/customers/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			getCustomer(w, r)
		case http.MethodPut:
			updateCustomer(w, r)
		case http.MethodDelete:
			deleteCustomer(w, r)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})


	log.Println("Server started on localhost port 3000")
	log.Fatal(http.ListenAndServe(":3000", mux))
}