package storage_test

import (
	"testing"
	"time"

	"library-system/models"
	"library-system/storage"
)

func TestBookCRUD(t *testing.T) {
	s := storage.NewMemoryStorage()

	book := &models.Book{
		ID:          "b1",
		ISBN:        "111-222-333",
		Title:       "Test Book",
		Author:      "Test Author",
		Category:    "Test Cat",
		TotalCopies: 5,
		AvailCopies: 5,
	}

	// Create
	if err := s.CreateBook(book); err != nil {
		t.Fatalf("unexpected error creating book: %v", err)
	}

	// Duplicate ISBN
	book2 := &models.Book{
		ID:          "b2",
		ISBN:        "111-222-333",
		Title:       "Test Book 2",
		Author:      "Test Author 2",
		Category:    "Test Cat",
		TotalCopies: 1,
		AvailCopies: 1,
	}
	if err := s.CreateBook(book2); err != storage.ErrAlreadyExists {
		t.Fatalf("expected ErrAlreadyExists, got %v", err)
	}

	// Read
	b, err := s.GetBook("b1")
	if err != nil || b.Title != "Test Book" {
		t.Fatalf("failed to retrieve book: %v", err)
	}

	// Update
	b.Title = "Updated Book Title"
	if err := s.UpdateBook(b); err != nil {
		t.Fatalf("failed to update book: %v", err)
	}

	updated, _ := s.GetBook("b1")
	if updated.Title != "Updated Book Title" {
		t.Fatalf("expected updated title, got %s", updated.Title)
	}

	// Search
	results, err := s.ListBooks("Updated", "Test Cat")
	if err != nil || len(results) != 1 {
		t.Fatalf("expected 1 result, got %d, err: %v", len(results), err)
	}

	// Delete
	if err := s.DeleteBook("b1"); err != nil {
		t.Fatalf("failed to delete book: %v", err)
	}

	if _, err := s.GetBook("b1"); err != storage.ErrNotFound {
		t.Fatalf("expected ErrNotFound after deletion, got %v", err)
	}
}

func TestBorrowAndReturn(t *testing.T) {
	s := storage.NewMemoryStorage()

	user := &models.User{
		ID:       "u1",
		Username: "john",
		Name:     "John Doe",
		Email:    "john@example.com",
		Role:     "member",
	}
	_ = s.CreateUser(user)

	book := &models.Book{
		ID:          "b1",
		ISBN:        "123-456",
		Title:       "Golang in Action",
		Author:      "Author A",
		Category:    "Tech",
		TotalCopies: 1,
		AvailCopies: 1,
	}
	_ = s.CreateBook(book)

	// Borrow book
	record, err := s.BorrowBook("u1", "b1", 7*24*time.Hour)
	if err != nil {
		t.Fatalf("failed to borrow book: %v", err)
	}
	if record.Status != models.BorrowStatusActive {
		t.Fatalf("expected status ACTIVE, got %s", record.Status)
	}

	// Verify available copies decreased
	b, _ := s.GetBook("b1")
	if b.AvailCopies != 0 {
		t.Fatalf("expected 0 available copies, got %d", b.AvailCopies)
	}

	// Borrowing again should fail (no copies)
	user2 := &models.User{ID: "u2", Username: "jane", Name: "Jane", Email: "jane@example.com", Role: "member"}
	_ = s.CreateUser(user2)
	if _, err := s.BorrowBook("u2", "b1", 7*24*time.Hour); err != storage.ErrNoCopiesAvail {
		t.Fatalf("expected ErrNoCopiesAvail, got %v", err)
	}

	// Cannot delete book while borrowed
	if err := s.DeleteBook("b1"); err == nil {
		t.Fatalf("expected error deleting book with active borrows, got nil")
	}

	// Return book
	returned, err := s.ReturnBook(record.ID)
	if err != nil {
		t.Fatalf("failed to return book: %v", err)
	}
	if returned.Status != models.BorrowStatusReturned || returned.ReturnDate == nil {
		t.Fatalf("expected status RETURNED with return date")
	}

	// Verify copy restored
	b, _ = s.GetBook("b1")
	if b.AvailCopies != 1 {
		t.Fatalf("expected 1 available copy after return, got %d", b.AvailCopies)
	}
}
