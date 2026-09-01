package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"library-system/models"
	"library-system/storage"
)

type LibraryService struct {
	store storage.Storage
}

func NewLibraryService(store storage.Storage) *LibraryService {
	return &LibraryService{store: store}
}

// AddBook adds a new book to the library catalogue.
func (s *LibraryService) AddBook(isbn, title, author, category string, totalCopies int) (*models.Book, error) {
	isbn = strings.TrimSpace(isbn)
	title = strings.TrimSpace(title)
	author = strings.TrimSpace(author)
	if isbn == "" || title == "" || author == "" {
		return nil, errors.New("isbn, title, and author are required")
	}
	if totalCopies <= 0 {
		return nil, errors.New("total copies must be greater than 0")
	}

	bookID := fmt.Sprintf("bk_%d", time.Now().UnixNano())
	book := &models.Book{
		ID:          bookID,
		ISBN:        isbn,
		Title:       title,
		Author:      author,
		Category:    strings.TrimSpace(category),
		TotalCopies: totalCopies,
		AvailCopies: totalCopies,
	}

	if err := s.store.CreateBook(book); err != nil {
		return nil, err
	}
	return s.store.GetBook(book.ID)
}

// GetBook retrieves a book by ID.
func (s *LibraryService) GetBook(id string) (*models.Book, error) {
	return s.store.GetBook(id)
}

// UpdateBook updates book details or inventory.
func (s *LibraryService) UpdateBook(id, title, author, category string, totalCopies int) (*models.Book, error) {
	book, err := s.store.GetBook(id)
	if err != nil {
		return nil, err
	}

	if title != "" {
		book.Title = strings.TrimSpace(title)
	}
	if author != "" {
		book.Author = strings.TrimSpace(author)
	}
	if category != "" {
		book.Category = strings.TrimSpace(category)
	}
	if totalCopies > 0 {
		borrowed := book.TotalCopies - book.AvailCopies
		if totalCopies < borrowed {
			return nil, fmt.Errorf("cannot reduce total copies below currently borrowed count (%d)", borrowed)
		}
		book.AvailCopies = totalCopies - borrowed
		book.TotalCopies = totalCopies
	}

	if err := s.store.UpdateBook(book); err != nil {
		return nil, err
	}
	return s.store.GetBook(id)
}

// DeleteBook removes a book if not actively borrowed.
func (s *LibraryService) DeleteBook(id string) error {
	return s.store.DeleteBook(id)
}

// SearchBooks searches books by text query and/or category.
func (s *LibraryService) SearchBooks(query, category string) ([]*models.Book, error) {
	return s.store.ListBooks(query, category)
}

// RegisterUser registers a new member or admin.
func (s *LibraryService) RegisterUser(username, name, email, role string) (*models.User, error) {
	username = strings.TrimSpace(username)
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	if username == "" || name == "" || email == "" {
		return nil, errors.New("username, name, and email are required")
	}
	if role != "admin" {
		role = "member"
	}

	userID := fmt.Sprintf("usr_%d", time.Now().UnixNano())
	user := &models.User{
		ID:       userID,
		Username: username,
		Name:     name,
		Email:    email,
		Role:     role,
	}

	if err := s.store.CreateUser(user); err != nil {
		return nil, err
	}
	return s.store.GetUser(user.ID)
}

// GetUser retrieves user by ID.
func (s *LibraryService) GetUser(id string) (*models.User, error) {
	return s.store.GetUser(id)
}

// ListUsers lists all registered users.
func (s *LibraryService) ListUsers() ([]*models.User, error) {
	return s.store.ListUsers()
}

// BorrowBook handles borrowing a book by a user.
func (s *LibraryService) BorrowBook(userID, bookID string, days int) (*models.BorrowRecord, error) {
	if days <= 0 {
		days = 14
	}
	duration := time.Duration(days) * 24 * time.Hour
	return s.store.BorrowBook(userID, bookID, duration)
}

// ReturnBook handles returning a borrowed book.
func (s *LibraryService) ReturnBook(recordID string) (*models.BorrowRecord, error) {
	return s.store.ReturnBook(recordID)
}

// GetUserBorrowHistory returns borrow records for a user.
func (s *LibraryService) GetUserBorrowHistory(userID string, status models.BorrowStatus) ([]*models.BorrowRecord, error) {
	return s.store.ListBorrowRecords(userID, status)
}

// ListAllBorrowRecords returns all borrow records with optional status filter.
func (s *LibraryService) ListAllBorrowRecords(status models.BorrowStatus) ([]*models.BorrowRecord, error) {
	return s.store.ListBorrowRecords("", status)
}
