//go:build ignore

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://postgres.mtrgloqfsijrykjrckrb:B3nt02o12%21%3F%21%3F%21%3F@aws-1-us-east-1.pooler.supabase.com:6543/postgres"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer db.Close()

	userID := "00000000-0000-0000-0000-000000000001"
	email := "admin@test.com"
	role := "admin"

	// Delete if exists
	_, _ = db.Exec("DELETE FROM user_profiles WHERE id = $1", userID)

	// Insert admin user
	_, err = db.Exec(`
		INSERT INTO user_profiles (id, email, full_name, role, is_active, created_at, updated_at)
		VALUES ($1, $2, 'Admin User', $3, TRUE, NOW(), NOW())
	`, userID, email, role)

	if err != nil {
		log.Fatalf("Failed to insert admin: %v", err)
	}

	fmt.Printf("Admin user created: %s (%s)\n", email, role)
}
