package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

type Customer struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Role      string `json:"role"`
	Email     string `json:"email"`
	Phone     int    `json:"phone"`
	Contacted bool   `json:"contacted"`
}


var customers = make(map[string]Customer)

func generateCustomerId() string {
	return uuid.New().String()
}

// gets the ID in the /customers/{id} path
func parseCustomerId(r *http.Request) string {
	if v := r.PathValue("id"); v != "" {
		return v
	}
	// for tests
	path := strings.TrimSuffix(r.URL.Path, "/")
	if rest, ok := strings.CutPrefix(path, "/customers/"); ok && rest != "" {
		return rest
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

	customers[c1.ID] = c1
	customers[c2.ID] = c2
	customers[c3.ID] = c3
}

func getCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	id := parseCustomerId(r)
	if id == "" {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "Customer not found"})
		return
	}

	c, ok := customers[id]

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

	list := toSlice(customers)

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

	customers[c.ID] = c

	w.WriteHeader(http.StatusCreated) 
	json.NewEncoder(w).Encode(c)
}

func updateCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	id := parseCustomerId(r)
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

	c, ok := customers[id]
	if !ok {
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

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(c)
}

func deleteCustomer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	id := parseCustomerId(r)
	if id == "" {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "customer not found"})
		return
	}

	_, ok := customers[id]
	if !ok {
		w.WriteHeader(http.StatusNotFound) 
		json.NewEncoder(w).Encode(nil)
		return
	}

	delete(customers, id)
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"deleted": id})
}

func home(w http.ResponseWriter, r *http.Request) {
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
}

func main() {
	seed()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", home)
	mux.HandleFunc("GET /customers", getCustomers)
	mux.HandleFunc("POST /customers", addCustomer)
	mux.HandleFunc("GET /customers/{id}", getCustomer)
	mux.HandleFunc("PUT /customers/{id}", updateCustomer)
	mux.HandleFunc("DELETE /customers/{id}", deleteCustomer)

	log.Println("Server started on localhost port 3000")
	log.Fatal(http.ListenAndServe(":3000", mux))
}
