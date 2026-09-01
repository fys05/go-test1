package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"library-system/models"
	"library-system/service"
	"library-system/storage"
)

type Server struct {
	svc *service.LibraryService
	mux *http.ServeMux
}

func NewServer(svc *service.LibraryService) *Server {
	s := &Server{
		svc: svc,
		mux: http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if data != nil {
		_ = json.NewEncoder(w).Encode(data)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func (s *Server) routes() {
	s.mux.HandleFunc("/healthz", s.handleHealth)
	s.mux.HandleFunc("/api/books", s.handleBooks)
	s.mux.HandleFunc("/api/books/", s.handleBookByID)
	s.mux.HandleFunc("/api/users", s.handleUsers)
	s.mux.HandleFunc("/api/users/", s.handleUserByID)
	s.mux.HandleFunc("/api/borrow", s.handleBorrow)
	s.mux.HandleFunc("/api/return", s.handleReturn)
	s.mux.HandleFunc("/api/records", s.handleRecords)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Book handlers
func (s *Server) handleBooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		query := r.URL.Query().Get("q")
		category := r.URL.Query().Get("category")
		books, err := s.svc.SearchBooks(query, category)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if books == nil {
			books = []*models.Book{}
		}
		writeJSON(w, http.StatusOK, books)

	case http.MethodPost:
		var req struct {
			ISBN        string `json:"isbn"`
			Title       string `json:"title"`
			Author      string `json:"author"`
			Category    string `json:"category"`
			TotalCopies int    `json:"total_copies"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		book, err := s.svc.AddBook(req.ISBN, req.Title, req.Author, req.Category, req.TotalCopies)
		if err != nil {
			if err == storage.ErrAlreadyExists {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, book)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleBookByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/books/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "book id required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		book, err := s.svc.GetBook(id)
		if err != nil {
			if err == storage.ErrNotFound {
				writeError(w, http.StatusNotFound, "book not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, book)

	case http.MethodPut:
		var req struct {
			Title       string `json:"title"`
			Author      string `json:"author"`
			Category    string `json:"category"`
			TotalCopies int    `json:"total_copies"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		book, err := s.svc.UpdateBook(id, req.Title, req.Author, req.Category, req.TotalCopies)
		if err != nil {
			if err == storage.ErrNotFound {
				writeError(w, http.StatusNotFound, "book not found")
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, book)

	case http.MethodDelete:
		err := s.svc.DeleteBook(id)
		if err != nil {
			if err == storage.ErrNotFound {
				writeError(w, http.StatusNotFound, "book not found")
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"message": "book deleted successfully"})

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// User handlers
func (s *Server) handleUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		users, err := s.svc.ListUsers()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if users == nil {
			users = []*models.User{}
		}
		writeJSON(w, http.StatusOK, users)

	case http.MethodPost:
		var req struct {
			Username string `json:"username"`
			Name     string `json:"name"`
			Email    string `json:"email"`
			Role     string `json:"role"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		user, err := s.svc.RegisterUser(req.Username, req.Name, req.Email, req.Role)
		if err != nil {
			if err == storage.ErrAlreadyExists {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, user)

	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

func (s *Server) handleUserByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/users/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "user id required")
		return
	}

	switch r.Method {
	case http.MethodGet:
		user, err := s.svc.GetUser(id)
		if err != nil {
			if err == storage.ErrNotFound {
				writeError(w, http.StatusNotFound, "user not found")
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, user)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
	}
}

// Borrow / Return handlers
func (s *Server) handleBorrow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		UserID string `json:"user_id"`
		BookID string `json:"book_id"`
		Days   int    `json:"days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	record, err := s.svc.BorrowBook(req.UserID, req.BookID, req.Days)
	if err != nil {
		if err == storage.ErrNotFound {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, record)
}

func (s *Server) handleReturn(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	var req struct {
		RecordID string `json:"record_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	record, err := s.svc.ReturnBook(req.RecordID)
	if err != nil {
		if err == storage.ErrNotFound {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, record)
}

func (s *Server) handleRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := r.URL.Query().Get("user_id")
	status := models.BorrowStatus(r.URL.Query().Get("status"))

	var records []*models.BorrowRecord
	var err error
	if userID != "" {
		records, err = s.svc.GetUserBorrowHistory(userID, status)
	} else {
		records, err = s.svc.ListAllBorrowRecords(status)
	}

	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if records == nil {
		records = []*models.BorrowRecord{}
	}
	writeJSON(w, http.StatusOK, records)
}
