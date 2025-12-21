package main

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	companyID := "2e153503-2e34-4785-9fab-54d3f6851f0e"

	// Reset companies table
	_, err = db.Exec(`
		UPDATE companies SET
			foundation_year = NULL,
			legal_name = NULL,
			headquarters = NULL,
			employees_range = NULL,
			trade_name = NULL,
			phone = NULL,
			email = NULL,
			cnae_primary = NULL,
			cnae_codes = '[]',
			capital_social = NULL,
			partners = '[]',
			cnpj_verified = false,
			linkedin_url = NULL,
			twitter_handle = NULL,
			instagram_url = NULL,
			facebook_url = NULL,
			key_executives = '[]',
			business_model = NULL,
			sector = NULL,
			main_products = '[]',
			pricing_model = NULL,
			target_audience = NULL,
			value_proposition = NULL,
			customer_segments = '[]',
			unique_selling_points = '[]',
			geographic_regions = '[]',
			competitors = '[]',
			competitor_details = '[]',
			industry_growth_rate = NULL,
			industry_trends = '[]',
			market_concentration = NULL,
			regulatory_context = NULL,
			market_share_status = NULL,
			recent_news = '[]',
			enrichment_sources = '[]',
			enrichment_status = 'pending',
			enrichment_completed_at = NULL,
			enrichment_error = NULL,
			updated_at = NOW()
		WHERE id = $1
	`, companyID)
	if err != nil {
		log.Fatalf("Failed to reset companies: %v", err)
	}
	fmt.Println("✓ Reset companies table")

	// Reset company_enrichment table
	_, err = db.Exec(`
		UPDATE company_enrichment SET
			step1_status = 'pending',
			step1_data = NULL,
			step1_error = NULL,
			step1_completed_at = NULL,
			step2_status = 'pending',
			step2_data = NULL,
			step2_error = NULL,
			step2_completed_at = NULL,
			step3_status = 'pending',
			step3_data = NULL,
			step3_error = NULL,
			step3_completed_at = NULL,
			updated_at = NOW()
		WHERE company_id = $1
	`, companyID)
	if err != nil {
		log.Fatalf("Failed to reset company_enrichment: %v", err)
	}
	fmt.Println("✓ Reset company_enrichment table")

	fmt.Println("\nKoro Sports enrichment data has been reset!")
	fmt.Println("You can now trigger Step 1 enrichment from the admin UI.")
}
