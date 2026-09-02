package storage

import (
	"errors"
	"strings"
	"sync"
	"time"

	"library-system/models"
)

var (
	ErrNotFound        = errors.New("record not found")
	ErrAlreadyExists   = errors.New("record already exists")
	ErrNoCopiesAvail   = errors.New("no copies available for borrow")
	ErrBookNotBorrowed = errors.New("book is not actively borrowed by user")
)

type Storage interface {
	// Book operations
	CreateBook(book *models.Book) error
	GetBook(id string) (*models.Book, error)
	GetBookByISBN(isbn string) (*models.Book, error)
	UpdateBook(book *models.Book) error
	DeleteBook(id string) error
	ListBooks(query string, category string) ([]*models.Book, error)

	// User operations
	CreateUser(user *models.User) error
	GetUser(id string) (*models.User, error)
	GetUserByUsername(username string) (*models.User, error)
	ListUsers() ([]*models.User, error)

	// Borrow operations
	BorrowBook(userID, bookID string, borrowDuration time.Duration) (*models.BorrowRecord, error)
	ReturnBook(recordID string) (*models.BorrowRecord, error)
	ListBorrowRecords(userID string, status models.BorrowStatus) ([]*models.BorrowRecord, error)
	GetBorrowRecord(id string) (*models.BorrowRecord, error)
}

type MemoryStorage struct {
	mu            sync.RWMutex
	books         map[string]*models.Book
	users         map[string]*models.User
	borrowRecords map[string]*models.BorrowRecord
}

func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		books:         make(map[string]*models.Book),
		users:         make(map[string]*models.User),
		borrowRecords: make(map[string]*models.BorrowRecord),
	}
}

func (s *MemoryStorage) CreateBook(book *models.Book) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.books[book.ID]; exists {
		return ErrAlreadyExists
	}
	for _, b := range s.books {
		if b.ISBN == book.ISBN {
			return ErrAlreadyExists
		}
	}
	now := time.Now().UTC()
	book.CreatedAt = now
	book.UpdatedAt = now
	bookCopy := *book
	s.books[book.ID] = &bookCopy
	return nil
}

func (s *MemoryStorage) GetBook(id string) (*models.Book, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	book, exists := s.books[id]
	if !exists {
		return nil, ErrNotFound
	}
	copy := *book
	return &copy, nil
}

func (s *MemoryStorage) GetBookByISBN(isbn string) (*models.Book, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, b := range s.books {
		if b.ISBN == isbn {
			copy := *b
			return &copy, nil
		}
	}
	return nil, ErrNotFound
}

func (s *MemoryStorage) UpdateBook(book *models.Book) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	curr, exists := s.books[book.ID]
	if !exists {
		return ErrNotFound
	}
	for _, b := range s.books {
		if b.ISBN == book.ISBN && b.ID != book.ID {
			return ErrAlreadyExists
		}
	}
	book.CreatedAt = curr.CreatedAt
	book.UpdatedAt = time.Now().UTC()
	bookCopy := *book
	s.books[book.ID] = &bookCopy
	return nil
}

func (s *MemoryStorage) DeleteBook(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.books[id]; !exists {
		return ErrNotFound
	}
	// Check if active borrows exist for this book
	for _, rec := range s.borrowRecords {
		if rec.BookID == id && rec.Status == models.BorrowStatusActive {
			return errors.New("cannot delete book with active borrow records")
		}
	}
	delete(s.books, id)
	return nil
}

func (s *MemoryStorage) ListBooks(query string, category string) ([]*models.Book, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	q := strings.ToLower(strings.TrimSpace(query))
	cat := strings.ToLower(strings.TrimSpace(category))

	var list []*models.Book
	for _, b := range s.books {
		if cat != "" && strings.ToLower(b.Category) != cat {
			continue
		}
		if q != "" {
			matchTitle := strings.Contains(strings.ToLower(b.Title), q)
			matchAuthor := strings.Contains(strings.ToLower(b.Author), q)
			matchISBN := strings.Contains(strings.ToLower(b.ISBN), q)
			if !matchTitle && !matchAuthor && !matchISBN {
				continue
			}
		}
		c := *b
		list = append(list, &c)
	}
	return list, nil
}

func (s *MemoryStorage) CreateUser(user *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[user.ID]; exists {
		return ErrAlreadyExists
	}
	for _, u := range s.users {
		if u.Username == user.Username || u.Email == user.Email {
			return ErrAlreadyExists
		}
	}
	user.CreatedAt = time.Now().UTC()
	userCopy := *user
	s.users[user.ID] = &userCopy
	return nil
}

func (s *MemoryStorage) GetUser(id string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[id]
	if !exists {
		return nil, ErrNotFound
	}
	c := *user
	return &c, nil
}

func (s *MemoryStorage) GetUserByUsername(username string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, u := range s.users {
		if u.Username == username {
			c := *u
			return &c, nil
		}
	}
	return nil, ErrNotFound
}

func (s *MemoryStorage) ListUsers() ([]*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var list []*models.User
	for _, u := range s.users {
		c := *u
		list = append(list, &c)
	}
	return list, nil
}

func (s *MemoryStorage) BorrowBook(userID, bookID string, duration time.Duration) (*models.BorrowRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[userID]; !exists {
		return nil, ErrNotFound
	}
	book, exists := s.books[bookID]
	if !exists {
		return nil, ErrNotFound
	}
	if book.AvailCopies <= 0 {
		return nil, ErrNoCopiesAvail
	}

	// Check if already actively borrowing this book
	for _, r := range s.borrowRecords {
		if r.UserID == userID && r.BookID == bookID && r.Status == models.BorrowStatusActive {
			return nil, errors.New("user is already borrowing a copy of this book")
		}
	}

	now := time.Now().UTC()
	if duration <= 0 {
		duration = 14 * 24 * time.Hour // default 14 days
	}

	record := &models.BorrowRecord{
		ID:         "br_" + time.Now().Format("20060102150405.000000000"),
		UserID:     userID,
		BookID:     bookID,
		BorrowDate: now,
		DueDate:    now.Add(duration),
		Status:     models.BorrowStatusActive,
	}

	book.AvailCopies--
	book.UpdatedAt = now
	s.borrowRecords[record.ID] = record

	copyRecord := *record
	return &copyRecord, nil
}

func (s *MemoryStorage) ReturnBook(recordID string) (*models.BorrowRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.borrowRecords[recordID]
	if !exists {
		return nil, ErrNotFound
	}
	if record.Status == models.BorrowStatusReturned {
		return nil, errors.New("book already returned")
	}

	now := time.Now().UTC()
	record.ReturnDate = &now
	record.Status = models.BorrowStatusReturned

	if book, exists := s.books[record.BookID]; exists {
		if book.AvailCopies < book.TotalCopies {
			book.AvailCopies++
			book.UpdatedAt = now
		}
	}

	copyRecord := *record
	return &copyRecord, nil
}

func (s *MemoryStorage) ListBorrowRecords(userID string, status models.BorrowStatus) ([]*models.BorrowRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC()
	var list []*models.BorrowRecord
	for _, r := range s.borrowRecords {
		if userID != "" && r.UserID != userID {
			continue
		}

		currentStatus := r.Status
		if currentStatus == models.BorrowStatusActive && now.After(r.DueDate) {
			currentStatus = models.BorrowStatusOverdue
		}

		if status != "" && currentStatus != status {
			continue
		}

		c := *r
		c.Status = currentStatus
		list = append(list, &c)
	}
	return list, nil
}

func (s *MemoryStorage) GetBorrowRecord(id string) (*models.BorrowRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, exists := s.borrowRecords[id]
	if !exists {
		return nil, ErrNotFound
	}
	c := *record
	if c.Status == models.BorrowStatusActive && time.Now().UTC().After(c.DueDate) {
		c.Status = models.BorrowStatusOverdue
	}
	return &c, nil
}
