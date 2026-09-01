package models

import "time"

// Book represents a library book.
type Book struct {
	ID          string    `json:"id"`
	ISBN        string    `json:"isbn"`
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Category    string    `json:"category"`
	TotalCopies int       `json:"total_copies"`
	AvailCopies int       `json:"available_copies"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// User represents a library member or admin.
type User struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"` // "member" or "admin"
	CreatedAt time.Time `json:"created_at"`
}

// BorrowStatus indicates current state of a borrow record.
type BorrowStatus string

const (
	BorrowStatusActive   BorrowStatus = "ACTIVE"
	BorrowStatusReturned BorrowStatus = "RETURNED"
	BorrowStatusOverdue  BorrowStatus = "OVERDUE"
)

// BorrowRecord tracks a book loan transaction.
type BorrowRecord struct {
	ID         string       `json:"id"`
	UserID     string       `json:"user_id"`
	BookID     string       `json:"book_id"`
	BorrowDate time.Time    `json:"borrow_date"`
	DueDate    time.Time    `json:"due_date"`
	ReturnDate *time.Time   `json:"return_date,omitempty"`
	Status     BorrowStatus `json:"status"`
}
