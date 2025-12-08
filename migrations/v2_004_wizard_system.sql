-- =============================================================================
-- Migration: v2_004_wizard_system
-- Purpose: Enable step-by-step human-in-the-loop framework validation
-- Date: 2024-12-02
-- Replaces: 038_framework_wizard.sql
-- =============================================================================

-- Add wizard tracking columns to analyses table
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS current_step INTEGER DEFAULT 0;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS wizard_mode BOOLEAN DEFAULT true;
ALTER TABLE analyses ADD COLUMN IF NOT EXISTS steps_completed TEXT[] DEFAULT '{}';

-- Create table for individual framework step tracking
CREATE TABLE IF NOT EXISTS analysis_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analysis_id UUID NOT NULL REFERENCES analyses(id) ON DELETE CASCADE,
    step_number INTEGER NOT NULL,
    framework_code VARCHAR(50) NOT NULL,

    -- Status: pending | generating | generated | approved | failed
    status VARCHAR(20) DEFAULT 'pending',

    -- AI-generated output (matches framework JSON structure)
    output JSONB,

    -- Human refinement input (context injection, NOT direct edit)
    human_context TEXT,
    refinement_notes TEXT,

    -- Clarifying questions for this framework
    clarifying_questions JSONB,
    human_answers JSONB,

    -- Iteration tracking (how many times regenerated)
    iteration_count INTEGER DEFAULT 0,

    -- Error tracking
    error_message TEXT,

    -- Processing time for this step (ms)
    processing_time_ms INTEGER,

    -- Timestamps
    generated_at TIMESTAMPTZ,
    approved_at TIMESTAMPTZ,
    approved_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Each framework can only appear once per analysis
    UNIQUE(analysis_id, framework_code)
);

-- Indexes for efficient wizard queries
CREATE INDEX IF NOT EXISTS idx_analysis_steps_analysis ON analysis_steps(analysis_id);
CREATE INDEX IF NOT EXISTS idx_analysis_steps_status ON analysis_steps(status);
CREATE INDEX IF NOT EXISTS idx_analysis_steps_analysis_step ON analysis_steps(analysis_id, step_number);

COMMENT ON TABLE analysis_steps IS 'Tracks individual framework steps in the wizard. Each step requires human approval before advancing.';
COMMENT ON COLUMN analysis_steps.human_context IS 'Additional context from human to inject into AI regeneration. NOT direct text replacement.';
COMMENT ON COLUMN analysis_steps.status IS 'Step status: pending, generating, generated, approved, failed';
