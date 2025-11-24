-- Minimal mock PostgreSQL schema and seed data to exercise API flows.
-- This is intentionally lightweight and only includes columns referenced
-- in repository queries and handlers.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users (used by Auth middleware + profile lookups)
CREATE TABLE IF NOT EXISTS user_profiles (
    id UUID PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    full_name TEXT,
    role TEXT DEFAULT 'user',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Submissions
CREATE TABLE IF NOT EXISTS submissions (
    id UUID PRIMARY KEY,
    company_name TEXT NOT NULL,
    cnpj TEXT,
    company_website TEXT,
    company_industry TEXT,
    company_size TEXT,
    company_location TEXT,
    contact_name TEXT NOT NULL,
    contact_email TEXT NOT NULL,
    contact_phone TEXT,
    contact_position TEXT,
    target_market TEXT,
    annual_revenue_min DOUBLE PRECISION,
    annual_revenue_max DOUBLE PRECISION,
    funding_stage TEXT,
    business_challenge TEXT NOT NULL,
    additional_notes TEXT,
    linkedin_url TEXT,
    twitter_handle TEXT,
    status TEXT NOT NULL,
    user_id UUID REFERENCES user_profiles(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Enrichment
CREATE TABLE IF NOT EXISTS enrichments (
    id UUID PRIMARY KEY,
    submission_id UUID NOT NULL REFERENCES submissions(id),
    status TEXT NOT NULL,
    content JSONB DEFAULT '{}'::jsonb,
    current_step TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Analyses (versioned)
CREATE TABLE IF NOT EXISTS analyses (
    id UUID PRIMARY KEY,
    submission_id UUID NOT NULL REFERENCES submissions(id),
    version INT NOT NULL DEFAULT 1,
    status TEXT NOT NULL,
    content JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Reports
CREATE TABLE IF NOT EXISTS reports (
    id UUID PRIMARY KEY,
    submission_id UUID NOT NULL REFERENCES submissions(id),
    analysis_id UUID REFERENCES analyses(id),
    status TEXT NOT NULL,
    content_url TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Seed users
INSERT INTO user_profiles (id, email, full_name, role)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'admin@imensiah.com', 'Admin User', 'admin'),
    ('00000000-0000-0000-0000-000000000002', 'testuser@example.com', 'Test User', 'user')
ON CONFLICT (id) DO NOTHING;

-- Seed a submission owned by the test user
INSERT INTO submissions (
    id, company_name, contact_name, contact_email, business_challenge,
    status, user_id
) VALUES (
    '11111111-1111-1111-1111-111111111111',
    'MockCo',
    'Alice Example',
    'alice@example.com',
    'Grow revenue in LATAM',
    'received',
    '00000000-0000-0000-0000-000000000002'
) ON CONFLICT (id) DO NOTHING;

-- Seed enrichment in pending status
INSERT INTO enrichments (id, submission_id, status, current_step)
VALUES (
    '22222222-2222-2222-2222-222222222222',
    '11111111-1111-1111-1111-111111111111',
    'pending',
    'initial_research'
) ON CONFLICT (id) DO NOTHING;

-- Seed analysis in pending status (version 1)
INSERT INTO analyses (id, submission_id, version, status)
VALUES (
    '33333333-3333-3333-3333-333333333333',
    '11111111-1111-1111-1111-111111111111',
    1,
    'pending'
) ON CONFLICT (id) DO NOTHING;

-- Seed report placeholder
INSERT INTO reports (id, submission_id, analysis_id, status, content_url)
VALUES (
    '44444444-4444-4444-4444-444444444444',
    '11111111-1111-1111-1111-111111111111',
    '33333333-3333-3333-3333-333333333333',
    'draft',
    'https://example.com/mock-report.pdf'
) ON CONFLICT (id) DO NOTHING;

