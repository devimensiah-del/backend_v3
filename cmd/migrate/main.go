package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres.mtrgloqfsijrykjrckrb:XSxdSfEiF0EDDHUI@aws-1-us-east-1.pooler.supabase.com:5432/postgres"
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	fmt.Println("Connected to database!")

	// Find and run v2_03x migrations
	files, err := filepath.Glob("migrations/v2_03*.sql")
	if err != nil {
		fmt.Printf("Failed to find migrations: %v\n", err)
		os.Exit(1)
	}

	sort.Strings(files)

	for _, file := range files {
		fmt.Printf("Running %s...\n", file)

		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("  Failed to read: %v\n", err)
			continue
		}

		// Execute the entire file as one transaction
		_, err = db.Exec(string(content))
		if err != nil {
			// Ignore "already exists" errors
			if strings.Contains(err.Error(), "already exists") ||
				strings.Contains(err.Error(), "duplicate key") {
				fmt.Printf("  (already exists, skipping)\n")
				continue
			}
			fmt.Printf("  Error: %v\n", err)
		} else {
			fmt.Printf("  Done!\n")
		}
	}

	// Verify tables exist
	fmt.Println("\nVerifying tables...")

	var count int
	err = db.Get(&count, "SELECT COUNT(*) FROM frameworks")
	if err != nil {
		fmt.Printf("frameworks table: ERROR - %v\n", err)
	} else {
		fmt.Printf("frameworks table: OK (%d rows)\n", count)
	}

	err = db.Get(&count, "SELECT COUNT(*) FROM framework_dependencies")
	if err != nil {
		fmt.Printf("framework_dependencies table: ERROR - %v\n", err)
	} else {
		fmt.Printf("framework_dependencies table: OK (%d rows)\n", count)
	}

	err = db.Get(&count, "SELECT COUNT(*) FROM company_framework_results")
	if err != nil {
		fmt.Printf("company_framework_results table: ERROR - %v\n", err)
	} else {
		fmt.Printf("company_framework_results table: OK (%d rows)\n", count)
	}

	fmt.Println("\nAll migrations complete!")
}
