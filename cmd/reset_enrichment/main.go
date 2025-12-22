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

	// Get company ID from args or use default (Koro)
	companyID := "2e153503-2e34-4785-9fab-54d3f6851f0e" // Koro
	if len(os.Args) > 1 {
		companyID = os.Args[1]
	}

	fmt.Printf("Resetting enrichment for company: %s\n", companyID)

	// Reset company_core table (enrichment status only)
	_, err = db.Exec(`
		UPDATE company_core SET
			enrichment_status = 'pending',
			enrichment_completed_at = NULL,
			enrichment_error = NULL,
			updated_at = NOW()
		WHERE id = $1
	`, companyID)
	if err != nil {
		log.Fatalf("Failed to reset company_core: %v", err)
	}
	fmt.Println("✓ Reset company_core table")

	// Reset company_step1_data table
	_, err = db.Exec(`
		UPDATE company_step1_data SET
			legal_name = NULL,
			trade_name = NULL,
			foundation_year = NULL,
			headquarters = NULL,
			employees_range = NULL,
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
			enrichment_sources = '[]',
			confidence_score = NULL,
			completed_at = NULL,
			updated_at = NOW()
		WHERE company_id = $1
	`, companyID)
	if err != nil {
		log.Fatalf("Failed to reset company_step1_data: %v", err)
	}
	fmt.Println("✓ Reset company_step1_data table")

	// Reset company_step2_data table
	_, err = db.Exec(`
		UPDATE company_step2_data SET
			business_model = NULL,
			sector = NULL,
			pricing_model = NULL,
			target_audience = NULL,
			value_proposition = NULL,
			main_products = '[]',
			customer_segments = '[]',
			unique_selling_points = '[]',
			geographic_regions = '[]',
			service_areas = '[]',
			enrichment_sources = '[]',
			confidence_score = NULL,
			completed_at = NULL,
			updated_at = NOW()
		WHERE company_id = $1
	`, companyID)
	if err != nil {
		log.Fatalf("Failed to reset company_step2_data: %v", err)
	}
	fmt.Println("✓ Reset company_step2_data table")

	// Reset company_step3_data table
	_, err = db.Exec(`
		UPDATE company_step3_data SET
			competitors = '[]',
			competitor_details = '[]',
			competitive_advantage = NULL,
			market_share = NULL,
			market_share_status = NULL,
			market_concentration = NULL,
			industry_growth_rate = NULL,
			industry_trends = '[]',
			regulatory_context = NULL,
			strengths = '[]',
			weaknesses = '[]',
			opportunities = '[]',
			threats = '[]',
			strategic_challenges = '[]',
			recent_news = '[]',
			tam_estimate = NULL,
			sam_estimate = NULL,
			som_estimate = NULL,
			enrichment_sources = '[]',
			confidence_score = NULL,
			completed_at = NULL,
			updated_at = NOW()
		WHERE company_id = $1
	`, companyID)
	if err != nil {
		log.Fatalf("Failed to reset company_step3_data: %v", err)
	}
	fmt.Println("✓ Reset company_step3_data table")

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

	fmt.Println("\nEnrichment data has been reset!")
	fmt.Println("You can now trigger Step 1 enrichment from the admin UI.")
}
