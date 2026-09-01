package main

import (
	"log"
	"net/http"
	"os"

	"library-system/api"
	"library-system/service"
	"library-system/storage"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	store := storage.NewMemoryStorage()
	svc := service.NewLibraryService(store)

	// Seed some initial data for convenience
	seedData(svc)

	server := api.NewServer(svc)

	log.Printf("Library Management System starting on port %s...", port)
	if err := http.ListenAndServe(":"+port, server.Handler()); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func seedData(svc *service.LibraryService) {
	// Seed admin and member
	admin, err := svc.RegisterUser("admin", "System Administrator", "admin@library.com", "admin")
	if err == nil {
		log.Printf("Seeded admin user: %s (%s)", admin.Username, admin.ID)
	}
	member, err := svc.RegisterUser("alice", "Alice Smith", "alice@example.com", "member")
	if err == nil {
		log.Printf("Seeded member user: %s (%s)", member.Username, member.ID)
	}

	// Seed books
	b1, _ := svc.AddBook("978-0134190440", "The Go Programming Language", "Alan Donovan, Brian Kernighan", "Programming", 3)
	b2, _ := svc.AddBook("978-0132350884", "Clean Code", "Robert C. Martin", "Software Engineering", 2)
	b3, _ := svc.AddBook("978-0201616224", "The Pragmatic Programmer", "David Thomas, Andrew Hunt", "Software Engineering", 5)

	if b1 != nil && member != nil {
		_, _ = svc.BorrowBook(member.ID, b1.ID, 14)
	}
	if b2 != nil {
		_ = b2
	}
	if b3 != nil {
		_ = b3
	}
}
