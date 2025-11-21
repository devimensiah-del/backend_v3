-- Backend V3 Database Schema
-- Complete schema for submissions, enrichments, analyses, and reports

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ==================== SUBMISSIONS TABLE ====================
-- Entry point for the entire workflow
CREATE TABLE IF NOT EXISTS submissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Company Information
    company_name VARCHAR(255) NOT NULL,
    company_website VARCHAR(500),
    company_industry VARCHAR(255),
    company_size VARCHAR(100),
    company_location VARCHAR(255),

    -- Contact Information
    contact_name VARCHAR(255) NOT NULL,
    contact_email VARCHAR(255) NOT NULL,
    contact_phone VARCHAR(50),
    contact_position VARCHAR(255),

    -- Business Context
    target_market TEXT,
    annual_revenue_min DECIMAL(15,2),
    annual_revenue_max DECIMAL(15,2),
    funding_stage VARCHAR(100),

    -- Submission Details
    business_challenge TEXT NOT NULL,
    additional_notes TEXT,
    linkedin_url VARCHAR(500),
    twitter_handle VARCHAR(100),

    -- Workflow Status
    -- pending → enriching → enriched → analyzing → analyzed → generating_report → completed
    -- Also: enrichment_failed, analysis_failed, report_failed, failed
    status VARCHAR(50) NOT NULL DEFAULT 'pending',

    -- User association (nullable for public submissions)
    user_id UUID,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP WITH TIME ZONE,

    -- Indices
    CONSTRAINT valid_status CHECK (status IN (
        'pending', 'enriching', 'enriched', 'analyzing', 'analyzed',
        'generating_report', 'completed', 'enrichment_failed',
        'analysis_failed', 'report_failed', 'failed'
    ))
);

-- Indices for fast queries
CREATE INDEX idx_submissions_status ON submissions(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_submissions_email ON submissions(contact_email) WHERE deleted_at IS NULL;
CREATE INDEX idx_submissions_user_id ON submissions(user_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_submissions_created_at ON submissions(created_at DESC);

-- ==================== ENRICHMENTS TABLE ====================
-- Stores enriched data from external sources
CREATE TABLE IF NOT EXISTS enrichments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,

    -- Enriched data (JSONB for flexible schema)
    company_info JSONB,
    market_data JSONB,
    competitors JSONB,
    financial_data JSONB,
    news JSONB,
    social_media JSONB,
    technologies JSONB,
    key_people JSONB,
    funding JSONB,
    partnerships JSONB,
    products JSONB,

    -- Progress tracking
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    progress INTEGER NOT NULL DEFAULT 0, -- 0-100
    current_step TEXT,
    sources_status JSONB,
    sources_used TEXT[], -- Array of source names

    -- Quality metrics
    enrichment_score DECIMAL(5,2), -- 0-100

    -- Error handling
    error_message TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    max_retries INTEGER NOT NULL DEFAULT 3,

    -- Timing
    processing_time_ms BIGINT,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,

    -- Audit
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT valid_enrichment_status CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    CONSTRAINT valid_progress CHECK (progress >= 0 AND progress <= 100),
    CONSTRAINT valid_enrichment_score CHECK (enrichment_score IS NULL OR (enrichment_score >= 0 AND enrichment_score <= 100))
);

-- Indices
CREATE INDEX idx_enrichments_submission_id ON enrichments(submission_id);
CREATE INDEX idx_enrichments_status ON enrichments(status);
CREATE INDEX idx_enrichments_created_at ON enrichments(created_at DESC);

-- ==================== ANALYSES TABLE ====================
-- Stores 7 strategic framework analyses
CREATE TABLE IF NOT EXISTS analyses (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    enrichment_id UUID NOT NULL REFERENCES enrichments(id) ON DELETE CASCADE,

    -- 7 Strategic Frameworks (JSONB)
    swot JSONB, -- SWOT Analysis
    pestel JSONB, -- PESTEL Analysis
    porter JSONB, -- Porter's Five Forces
    okrs JSONB, -- Objectives and Key Results
    bcg_matrix JSONB, -- BCG Growth-Share Matrix
    value_proposition JSONB, -- Value Proposition Canvas
    business_model JSONB, -- Business Model Canvas

    -- Synthesis combining all frameworks
    synthesis JSONB,

    -- Status tracking
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    processing_time_ms BIGINT, -- Total parallel processing time

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT valid_analysis_status CHECK (status IN ('pending', 'processing', 'completed', 'failed'))
);

-- Indices
CREATE INDEX idx_analyses_submission_id ON analyses(submission_id);
CREATE INDEX idx_analyses_enrichment_id ON analyses(enrichment_id);
CREATE INDEX idx_analyses_status ON analyses(status);
CREATE INDEX idx_analyses_created_at ON analyses(created_at DESC);

-- ==================== REPORTS TABLE ====================
-- Stores 13 HTML pages and PDF URL
CREATE TABLE IF NOT EXISTS reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    submission_id UUID NOT NULL REFERENCES submissions(id) ON DELETE CASCADE,
    analysis_id UUID NOT NULL REFERENCES analyses(id) ON DELETE CASCADE,

    -- 13 HTML Pages (TEXT for large HTML content)
    cover_page TEXT,
    executive_summary TEXT,
    table_of_contents TEXT,
    swot_page TEXT,
    pestel_page TEXT,
    porter_page TEXT,
    okr_page TEXT,
    bcg_matrix_page TEXT,
    value_proposition_page TEXT,
    business_model_page TEXT,
    strategic_priorities_page TEXT,
    risks_mitigation_page TEXT,
    appendix_page TEXT,

    -- PDF Information
    pdf_url VARCHAR(1000),
    pdf_generated_at TIMESTAMP WITH TIME ZONE,
    pdf_generation_status VARCHAR(50) NOT NULL DEFAULT 'pending',

    -- Status tracking
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    generation_time_ms BIGINT,
    total_pages INTEGER NOT NULL DEFAULT 13,

    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,

    CONSTRAINT valid_report_status CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    CONSTRAINT valid_pdf_status CHECK (pdf_generation_status IN ('pending', 'processing', 'completed', 'failed'))
);

-- Indices
CREATE INDEX idx_reports_submission_id ON reports(submission_id);
CREATE INDEX idx_reports_analysis_id ON reports(analysis_id);
CREATE INDEX idx_reports_status ON reports(status);
CREATE INDEX idx_reports_created_at ON reports(created_at DESC);

-- ==================== HELPER FUNCTIONS ====================

-- Function to update updated_at timestamp automatically
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply update_updated_at trigger to all tables
CREATE TRIGGER update_submissions_updated_at BEFORE UPDATE ON submissions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_enrichments_updated_at BEFORE UPDATE ON enrichments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_analyses_updated_at BEFORE UPDATE ON analyses
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_reports_updated_at BEFORE UPDATE ON reports
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ==================== SAMPLE DATA (Optional) ====================

-- Uncomment to insert sample submission for testing
/*
INSERT INTO submissions (
    company_name, company_website, company_industry, company_location,
    contact_name, contact_email, business_challenge,
    status
) VALUES (
    'Acme Corporation',
    'https://acme.com',
    'Technology',
    'San Francisco, CA',
    'John Doe',
    'john@acme.com',
    'We need to scale our SaaS platform to handle 10x more users',
    'pending'
);
*/

-- ==================== SCHEMA DOCUMENTATION ====================

COMMENT ON TABLE submissions IS 'Company submissions - entry point for the workflow';
COMMENT ON TABLE enrichments IS 'Enriched external data from 15+ sources';
COMMENT ON TABLE analyses IS '7 strategic framework analyses processed in parallel';
COMMENT ON TABLE reports IS '13 HTML page reports with PDF generation';

COMMENT ON COLUMN analyses.processing_time_ms IS 'Time for parallel processing - should be 2.8-4.4x faster than sequential';
COMMENT ON COLUMN enrichments.enrichment_score IS 'Quality score 0-100 based on data completeness';
COMMENT ON COLUMN reports.total_pages IS 'Always 13 pages in the report';

-- ==================== PERFORMANCE NOTES ====================

/*
Expected Performance Metrics:
- Enrichment: 30-60 seconds (15+ API calls)
- Analysis: 5-10 seconds (7 frameworks in parallel)
- Report Generation: 2-5 seconds (13 HTML pages)
- Total workflow: ~40-75 seconds

Parallel Analysis Improvement:
- Sequential: ~15-20 seconds
- Parallel: ~5-10 seconds
- Speed improvement: 2.8-4.4x
*/
