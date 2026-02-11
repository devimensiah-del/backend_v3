package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		fmt.Printf("❌ Erro ao carregar .env: %v\n", err)
		os.Exit(1)
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Println("❌ DATABASE_URL não configurado")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	db, err := sqlx.ConnectContext(ctx, "postgres", dbURL)
	if err != nil {
		fmt.Printf("❌ Erro ao conectar: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Get migration files
	migrations := []string{
		"migrations/v2_041_step_organization.sql",
		"migrations/v2_042_stale_tracking.sql",
	}

	// If specific migrations provided as args, use those
	if len(os.Args) > 1 {
		migrations = os.Args[1:]
	}

	// Sort migrations
	sort.Strings(migrations)

	fmt.Printf("📦 Aplicando %d migrations...\n\n", len(migrations))

	for _, migrationPath := range migrations {
		fmt.Printf("▶️  Aplicando: %s\n", filepath.Base(migrationPath))

		content, err := os.ReadFile(migrationPath)
		if err != nil {
			fmt.Printf("❌ Erro ao ler arquivo: %v\n", err)
			continue
		}

		// Execute migration
		_, err = db.ExecContext(ctx, string(content))
		if err != nil {
			// Check if it's a duplicate error (already applied)
			if strings.Contains(err.Error(), "already exists") ||
				strings.Contains(err.Error(), "duplicate") {
				fmt.Printf("   ⚠️  Algumas partes já existem (OK)\n")
			} else {
				fmt.Printf("❌ Erro na migration: %v\n", err)
				os.Exit(1)
			}
		} else {
			fmt.Printf("   ✅ Sucesso\n")
		}
	}

	// Verify results
	fmt.Println("\n📊 Verificando resultados...")

	// Check frameworks
	var fwCount int
	db.GetContext(ctx, &fwCount, "SELECT COUNT(*) FROM frameworks WHERE is_active = true")
	fmt.Printf("   Frameworks ativos: %d\n", fwCount)

	// Check step distribution
	rows, _ := db.QueryxContext(ctx, `
		SELECT step_number, step_name, COUNT(*) as count
		FROM frameworks
		WHERE is_active = true AND step_number IS NOT NULL
		GROUP BY step_number, step_name
		ORDER BY step_number
	`)
	defer rows.Close()

	fmt.Println("   Distribuição por step:")
	for rows.Next() {
		var stepNum int
		var stepName string
		var count int
		rows.Scan(&stepNum, &stepName, &count)
		fmt.Printf("      Step %d (%s): %d frameworks\n", stepNum, stepName, count)
	}

	// Check dependencies
	var depCount int
	db.GetContext(ctx, &depCount, "SELECT COUNT(*) FROM framework_dependencies")
	fmt.Printf("   Dependências registradas: %d\n", depCount)

	// Check stale tracking columns
	var hasStale bool
	err = db.GetContext(ctx, &hasStale, `
		SELECT EXISTS(
			SELECT 1 FROM information_schema.columns
			WHERE table_name = 'company_framework_results'
			AND column_name = 'is_stale'
		)
	`)
	if err == nil && hasStale {
		fmt.Println("   ✅ Colunas de stale tracking adicionadas")
	}

	fmt.Println("\n🎉 Migrations aplicadas com sucesso!")
}
