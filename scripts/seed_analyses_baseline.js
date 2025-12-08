/**
 * Seed script: Convert analyses_rows.json to baseline SQL INSERT statements
 *
 * Usage:
 *   node scripts/seed_analyses_baseline.js > migrations/seed_analyses_data.sql
 *
 * Or to run directly against DB:
 *   node scripts/seed_analyses_baseline.js | psql -d your_database
 */

const fs = require('fs');
const path = require('path');

// Path to the JSON file
const JSON_FILE = process.argv[2] || 'C:\\Users\\pradord\\Downloads\\analyses_rows.json';

// Read and parse JSON
const data = JSON.parse(fs.readFileSync(JSON_FILE, 'utf8'));

console.log('-- =============================================================================');
console.log('-- SEED SCRIPT: analyses table data for BASELINE schema');
console.log('-- Generated:', new Date().toISOString());
console.log('-- Source:', JSON_FILE);
console.log('-- Row count:', data.length);
console.log('-- =============================================================================');
console.log('');
console.log('BEGIN;');
console.log('');

// Helper to escape strings for SQL
function sqlEscape(val) {
  if (val === null || val === undefined) return 'NULL';
  if (typeof val === 'boolean') return val ? 'true' : 'false';
  if (typeof val === 'number') return val.toString();

  // For strings, escape single quotes by doubling them
  const str = String(val).replace(/'/g, "''");
  return `'${str}'`;
}

// Helper for JSONB - parse if string, then stringify properly
function jsonbValue(val) {
  if (val === null || val === undefined) return 'NULL';

  // If it's already a string (JSON stringified), use it directly
  if (typeof val === 'string') {
    // Escape single quotes for SQL
    const escaped = val.replace(/'/g, "''");
    return `'${escaped}'::jsonb`;
  }

  // If it's an object, stringify it
  const str = JSON.stringify(val).replace(/'/g, "''");
  return `'${str}'::jsonb`;
}

// Helper for timestamp
function tsValue(val) {
  if (val === null || val === undefined) return 'NULL';
  return `'${val}'::timestamptz`;
}

// Framework columns in baseline schema
const frameworkCols = [
  'swot', 'pestel', 'porter', 'okrs', 'synthesis',
  'tam_sam_som', 'benchmarking', 'blue_ocean',
  'growth_hacking', 'scenarios', 'bsc', 'decision_matrix'
];

// Process each row
data.forEach((row, idx) => {
  // Build column list and values
  const cols = [];
  const vals = [];

  // Required columns
  cols.push('id'); vals.push(sqlEscape(row.id));
  cols.push('submission_id'); vals.push(sqlEscape(row.submission_id));
  cols.push('enrichment_id'); vals.push(sqlEscape(row.enrichment_id));
  cols.push('company_id'); vals.push(sqlEscape(row.company_id));

  // Framework columns (JSONB)
  frameworkCols.forEach(col => {
    cols.push(col);
    vals.push(jsonbValue(row[col]));
  });

  // Status and workflow
  cols.push('status'); vals.push(sqlEscape(row.status || 'pending'));
  cols.push('error_message'); vals.push(sqlEscape(row.error_message));
  cols.push('processing_time_ms'); vals.push(row.processing_time_ms ?? 'NULL');

  // Visibility
  cols.push('is_visible_to_user'); vals.push(row.is_visible_to_user ?? false);
  cols.push('is_blurred'); vals.push(row.is_blurred ?? true);
  cols.push('is_public'); vals.push(row.is_public ?? false);
  cols.push('access_code'); vals.push(sqlEscape(row.access_code));
  cols.push('access_code_created_at'); vals.push(tsValue(row.access_code_created_at));

  // PDF
  cols.push('pdf_url'); vals.push(sqlEscape(row.pdf_url));
  cols.push('pdf_generated_at'); vals.push(tsValue(row.pdf_generated_at));

  // Approval workflow
  cols.push('approved_at'); vals.push(tsValue(row.approved_at));
  cols.push('approved_by'); vals.push(sqlEscape(row.approved_by));
  cols.push('sent_at'); vals.push(tsValue(row.sent_at));
  cols.push('sent_to'); vals.push(sqlEscape(row.sent_to));

  // Timestamps
  cols.push('created_at'); vals.push(tsValue(row.created_at) || 'NOW()');
  cols.push('updated_at'); vals.push(tsValue(row.updated_at) || 'NOW()');
  cols.push('completed_at'); vals.push(tsValue(row.completed_at));
  cols.push('deleted_at'); vals.push(tsValue(row.deleted_at));

  console.log(`-- Row ${idx + 1}/${data.length}: ${row.id}`);
  console.log(`INSERT INTO analyses (${cols.join(', ')})`);
  console.log(`VALUES (${vals.join(', ')})`);
  console.log('ON CONFLICT (id) DO UPDATE SET');

  // Generate update clause for all columns except id
  const updates = cols.slice(1).map((col, i) => `  ${col} = EXCLUDED.${col}`);
  console.log(updates.join(',\n') + ';');
  console.log('');
});

console.log('COMMIT;');
console.log('');
console.log('-- =============================================================================');
console.log('-- Verification');
console.log('-- =============================================================================');
console.log('-- SELECT COUNT(*) FROM analyses;');
console.log(`-- Expected: ${data.length}`);
