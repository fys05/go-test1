package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"library-system/api"
	"library-system/models"
	"library-system/service"
	"library-system/storage"
)

func setupTestServer() *api.Server {
	store := storage.NewMemoryStorage()
	svc := service.NewLibraryService(store)
	return api.NewServer(svc)
}

func TestHealthCheck(t *testing.T) {
	server := setupTestServer()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestBookAPI(t *testing.T) {
	server := setupTestServer()

	// 1. Create Book
	bookPayload := map[string]interface{}{
		"isbn":         "978-1234567890",
		"title":        "Microservices in Go",
		"author":       "Jane Developer",
		"category":     "Backend",
		"total_copies": 3,
	}
	body, _ := json.Marshal(bookPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
	}

	var createdBook models.Book
	if err := json.NewDecoder(rec.Body).Decode(&createdBook); err != nil {
		t.Fatalf("failed to decode book: %v", err)
	}

	// 2. Get Book by ID
	req = httptest.NewRequest(http.MethodGet, "/api/books/"+createdBook.ID, nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	// 3. Search Books
	req = httptest.NewRequest(http.MethodGet, "/api/books?q=Microservices", nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}
	var books []*models.Book
	_ = json.NewDecoder(rec.Body).Decode(&books)
	if len(books) != 1 {
		t.Fatalf("expected 1 book returned, got %d", len(books))
	}
}

func TestBorrowAndReturnAPI(t *testing.T) {
	server := setupTestServer()

	// Create user
	userPayload := map[string]interface{}{
		"username": "bob",
		"name":     "Bob",
		"email":    "bob@example.com",
		"role":     "member",
	}
	body, _ := json.Marshal(userPayload)
	req := httptest.NewRequest(http.MethodPost, "/api/users", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	var user models.User
	_ = json.NewDecoder(rec.Body).Decode(&user)

	// Create book
	bookPayload := map[string]interface{}{
		"isbn":         "978-0000000001",
		"title":        "Concurrency in Go",
		"author":       "Katherine Cox-Buday",
		"category":     "Programming",
		"total_copies": 1,
	}
	body, _ = json.Marshal(bookPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	var book models.Book
	_ = json.NewDecoder(rec.Body).Decode(&book)

	// Borrow book
	borrowPayload := map[string]interface{}{
		"user_id": user.ID,
		"book_id": book.ID,
		"days":    7,
	}
	body, _ = json.Marshal(borrowPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/borrow", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for borrow, got %d: %s", rec.Code, rec.Body.String())
	}
	var record models.BorrowRecord
	_ = json.NewDecoder(rec.Body).Decode(&record)

	// List borrow records
	req = httptest.NewRequest(http.MethodGet, "/api/records?user_id="+user.ID, nil)
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	var records []*models.BorrowRecord
	_ = json.NewDecoder(rec.Body).Decode(&records)
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}

	// Return book
	returnPayload := map[string]interface{}{
		"record_id": record.ID,
	}
	body, _ = json.Marshal(returnPayload)
	req = httptest.NewRequest(http.MethodPost, "/api/return", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for return, got %d: %s", rec.Code, rec.Body.String())
	}
}
