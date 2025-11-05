package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

type Customer struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	Email     string `json:"email"`
	Phone     int    `json:"phone"`
	Contacted bool   `json:"contacted"`
}

var (
	dbMu      sync.RWMutex
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

func addCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "unable to read body"})
		return
	}
	defer r.Body.Close()

	// Accept Customer payload without an id
	var incomingCustomer struct {
	  Name      string `json:"name"`
		Role      string `json:"role"`
		Email     string `json:"email"`
		Phone     any    `json:"phone"` // tests send a number; allow string or number
		Contacted bool   `json:"contacted"`
	}

	if err := json.Unmarshal(body, &incomingCustomer); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}

	// Normalize phone to int
	phoneInt := 0
	switch v := incomingCustomer.Phone.(type) {
	case float64:
		phoneInt = int(v)
	case string:
		if p, err := strconv.Atoi(v); err == nil {
			phoneInt = p
		}
	}

	c := Customer{
		ID:        generateCustomerId(),
		Name:      incomingCustomer.Name,
		Role:      incomingCustomer.Role,
		Email:     incomingCustomer.Email,
		Phone:     phoneInt,
		Contacted: incomingCustomer.Contacted,
	}

	dbMu.Lock()
	customers[c.ID] = c
	dbMu.Unlock()

	w.WriteHeader(http.StatusCreated) 
	json.NewEncoder(w).Encode(c)
}

func updateCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	id := parseCustomerId(r.URL.Path)
	if id == "" {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "customer not found"})
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "unable to read request body"})
		return
	}
	defer r.Body.Close()

	// Allow partial updates (don't touch fields that aren't included in PUT)
	var patch map[string]any
	if err := json.Unmarshal(body, &patch); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON"})
		return
	}

	dbMu.Lock()
	c, ok := customers[id]
	if !ok {
		dbMu.Unlock()
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(nil)
		return
	}

	if v, ok := patch["name"].(string); ok {
		c.Name = v
	}
	if v, ok := patch["role"].(string); ok {
		c.Role = v
	}
	if v, ok := patch["email"].(string); ok {
		c.Email = v
	}
	if v, ok := patch["phone"]; ok {
		switch pv := v.(type) {
		case float64:
			c.Phone = int(pv)
		case string:
			if p, err := strconv.Atoi(pv); err == nil {
				c.Phone = p
			}
		}
	}
	if v, ok := patch["contacted"].(bool); ok {
		c.Contacted = v
	}

	customers[id] = c
	dbMu.Unlock()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(c)
}

func deleteCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	id := parseCustomerId(r.URL.Path)
	if id == "" {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "customer not found"})
		return
	}

	dbMu.Lock()
	_, ok := customers[id]
	if !ok {
		dbMu.Unlock()
		w.WriteHeader(http.StatusNotFound) 
		json.NewEncoder(w).Encode(nil)
		return
	}
	delete(customers, id)
	dbMu.Unlock()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"deleted": id})
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
