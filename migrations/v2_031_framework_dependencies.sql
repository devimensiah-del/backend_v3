-- Migration: v2_031_framework_dependencies.sql
-- Description: Create framework_dependencies table for dependency resolution
-- Only stores DIRECT dependencies, transitive dependencies are resolved at runtime

CREATE TABLE IF NOT EXISTS framework_dependencies (
    id UUID NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,

    -- Relationship (only DIRECT dependencies, transitive resolved at runtime)
    framework_id UUID NOT NULL,               -- The framework that has the dependency
    depends_on_id UUID NOT NULL,              -- The framework it depends on

    -- Dependency type (for future flexibility)
    dependency_type VARCHAR(20) NOT NULL DEFAULT 'required',  -- 'required' | 'optional' | 'enhances'

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints
    CONSTRAINT fk_framework_dependencies_framework
        FOREIGN KEY (framework_id) REFERENCES frameworks(id) ON DELETE CASCADE,
    CONSTRAINT fk_framework_dependencies_depends_on
        FOREIGN KEY (depends_on_id) REFERENCES frameworks(id) ON DELETE CASCADE,
    CONSTRAINT uq_framework_dependencies UNIQUE (framework_id, depends_on_id),
    CONSTRAINT chk_no_self_dependency CHECK (framework_id != depends_on_id)
);

-- Indexes for dependency resolution
CREATE INDEX IF NOT EXISTS idx_framework_dependencies_framework_id ON framework_dependencies(framework_id);
CREATE INDEX IF NOT EXISTS idx_framework_dependencies_depends_on_id ON framework_dependencies(depends_on_id);

COMMENT ON TABLE framework_dependencies IS 'Direct dependencies between frameworks. Transitive dependencies resolved at runtime via topological sort.';
COMMENT ON COLUMN framework_dependencies.dependency_type IS 'required = must complete before this framework runs. optional = enhances results if available. enhances = uses output but not required.';
