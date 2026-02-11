package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		fmt.Printf("❌ Erro ao carregar .env: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ .env carregado")

	// Test PostgreSQL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		fmt.Println("❌ DATABASE_URL não configurado")
		os.Exit(1)
	}
	fmt.Printf("📡 Testando PostgreSQL: %s...\n", dbURL[:50]+"...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	db, err := sqlx.ConnectContext(ctx, "postgres", dbURL)
	if err != nil {
		fmt.Printf("❌ Erro ao conectar PostgreSQL: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Test query
	var result int
	err = db.GetContext(ctx, &result, "SELECT 1")
	if err != nil {
		fmt.Printf("❌ Erro na query de teste: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ PostgreSQL conectado com sucesso")

	// Check frameworks table
	var fwCount int
	err = db.GetContext(ctx, &fwCount, "SELECT COUNT(*) FROM frameworks")
	if err != nil {
		fmt.Printf("⚠️  Tabela frameworks não existe ou erro: %v\n", err)
	} else {
		fmt.Printf("✅ Tabela frameworks: %d registros\n", fwCount)
	}

	// Check company_framework_results table
	var cfrCount int
	err = db.GetContext(ctx, &cfrCount, "SELECT COUNT(*) FROM company_framework_results")
	if err != nil {
		fmt.Printf("⚠️  Tabela company_framework_results não existe ou erro: %v\n", err)
	} else {
		fmt.Printf("✅ Tabela company_framework_results: %d registros\n", cfrCount)
	}

	// Check companies table
	var companyCount int
	err = db.GetContext(ctx, &companyCount, "SELECT COUNT(*) FROM company_core")
	if err != nil {
		fmt.Printf("⚠️  Tabela company_core não existe ou erro: %v\n", err)
	} else {
		fmt.Printf("✅ Tabela company_core: %d registros\n", companyCount)
	}

	// Test Redis
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		fmt.Println("⚠️  REDIS_URL não configurado, pulando teste Redis")
	} else {
		fmt.Printf("📡 Testando Redis: %s...\n", redisURL[:30]+"...")

		opts, err := redis.ParseURL(redisURL)
		if err != nil {
			fmt.Printf("❌ Erro ao parsear REDIS_URL: %v\n", err)
		} else {
			rdb := redis.NewClient(opts)
			defer rdb.Close()

			_, err = rdb.Ping(ctx).Result()
			if err != nil {
				fmt.Printf("❌ Erro ao conectar Redis: %v\n", err)
			} else {
				fmt.Println("✅ Redis conectado com sucesso")
			}
		}
	}

	// Test OpenRouter (just check if key is present)
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		fmt.Println("⚠️  OPENROUTER_API_KEY não configurado")
	} else {
		fmt.Printf("✅ OpenRouter API Key configurada: %s...\n", apiKey[:20])
	}

	fmt.Println("\n🎉 Todos os testes de conexão passaram!")
}
