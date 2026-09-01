# Library Management System (Go)

A lightweight and robust Library Management System REST API built in Go.

## Features

- **Book Catalogue Management**: Add, update, view, delete, and search books by query (title/author/ISBN) and category.
- **User Management**: Register and view users with role-based attributes (`admin`, `member`).
- **Borrow & Return System**:
  - Book borrowing with inventory checking and configurable duration.
  - Book returning with automatic inventory restoration.
  - Borrow history and status tracking (`ACTIVE`, `RETURNED`, `OVERDUE`).
- **RESTful API**: Clean JSON REST API standard.
- **In-Memory Thread-Safe Storage**: Concurrent read/write safe repository layer.

## API Endpoints

### Health Check
- `GET /healthz` - Health status

### Books
- `GET /api/books` - List/search books (`?q=keyword&category=cat`)
- `POST /api/books` - Add a new book
- `GET /api/books/{id}` - Get book details
- `PUT /api/books/{id}` - Update book
- `DELETE /api/books/{id}` - Delete book (allowed only if not actively borrowed)

### Users
- `GET /api/users` - List all users
- `POST /api/users` - Register a new user
- `GET /api/users/{id}` - Get user details

### Borrow & Return
- `POST /api/borrow` - Borrow a book (`{"user_id": "...", "book_id": "...", "days": 14}`)
- `POST /api/return` - Return a book (`{"record_id": "..."}`)
- `GET /api/records` - Query borrow records (`?user_id=...&status=ACTIVE`)

## Getting Started

### Run Tests
```bash
go test -v ./...
```

### Run Server
```bash
go run main.go
```
The server will start on port `8080` (or the port specified in `PORT` env var).
