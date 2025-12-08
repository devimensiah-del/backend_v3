/**
 * Generate a single SQL file for direct psql connection
 * FKs set to NULL, FK checks disabled
 */

const fs = require('fs');

const JSON_FILE = 'C:\\Users\\pradord\\Downloads\\analyses_rows.json';
const OUTPUT_FILE = 'C:\\Users\\pradord\\Documents\\Projects\\imensiah_new\\backend_v3\\migrations\\seed_analyses_DIRECT.sql';

const data = JSON.parse(fs.readFileSync(JSON_FILE, 'utf8'));

function sqlEscape(val) {
  if (val === null || val === undefined) return 'NULL';
  if (typeof val === 'boolean') return val ? 'true' : 'false';
  if (typeof val === 'number') return val.toString();
  const str = String(val).replace(/'/g, "''");
  return `'${str}'`;
}

function jsonbValue(val) {
  if (val === null || val === undefined) return 'NULL';
  if (typeof val === 'string') {
    const escaped = val.replace(/'/g, "''");
    return `'${escaped}'::jsonb`;
  }
  const str = JSON.stringify(val).replace(/'/g, "''");
  return `'${str}'::jsonb`;
}

function tsValue(val) {
  if (val === null || val === undefined) return 'NULL';
  return `'${val}'::timestamptz`;
}

const frameworkCols = [
  'swot', 'pestel', 'porter', 'okrs', 'synthesis',
  'tam_sam_som', 'benchmarking', 'blue_ocean',
  'growth_hacking', 'scenarios', 'bsc', 'decision_matrix'
];

let sql = `-- =============================================================================
-- SEED ANALYSES - DIRECT CONNECTION (psql)
-- =============================================================================
-- Generated: ${new Date().toISOString()}
-- Rows: ${data.length}
--
-- Run via psql (NOT Supabase SQL Editor - file too large):
--   psql "postgresql://postgres:[PASSWORD]@[HOST]:5432/postgres" -f seed_analyses_DIRECT.sql
-- =============================================================================

-- Disable FK checks
SET session_replication_role = 'replica';

`;

data.forEach((row, idx) => {
  const cols = [];
  const vals = [];

  cols.push('id'); vals.push(sqlEscape(row.id));
  cols.push('submission_id'); vals.push('NULL');
  cols.push('enrichment_id'); vals.push('NULL');
  cols.push('company_id'); vals.push('NULL');

  frameworkCols.forEach(col => {
    cols.push(col);
    vals.push(jsonbValue(row[col]));
  });

  cols.push('status'); vals.push(sqlEscape(row.status || 'pending'));
  cols.push('error_message'); vals.push(sqlEscape(row.error_message));
  cols.push('processing_time_ms'); vals.push(row.processing_time_ms ?? 'NULL');
  cols.push('is_visible_to_user'); vals.push(row.is_visible_to_user ?? false);
  cols.push('is_blurred'); vals.push(row.is_blurred ?? true);
  cols.push('is_public'); vals.push(row.is_public ?? false);
  cols.push('access_code'); vals.push(sqlEscape(row.access_code));
  cols.push('access_code_created_at'); vals.push(tsValue(row.access_code_created_at));
  cols.push('pdf_url'); vals.push(sqlEscape(row.pdf_url));
  cols.push('pdf_generated_at'); vals.push(tsValue(row.pdf_generated_at));
  cols.push('approved_at'); vals.push(tsValue(row.approved_at));
  cols.push('approved_by'); vals.push(sqlEscape(row.approved_by));
  cols.push('sent_at'); vals.push(tsValue(row.sent_at));
  cols.push('sent_to'); vals.push(sqlEscape(row.sent_to));
  cols.push('created_at'); vals.push(tsValue(row.created_at) || 'NOW()');
  cols.push('updated_at'); vals.push(tsValue(row.updated_at) || 'NOW()');
  cols.push('completed_at'); vals.push(tsValue(row.completed_at));
  cols.push('deleted_at'); vals.push(tsValue(row.deleted_at));

  sql += `-- Row ${idx + 1}/${data.length}: ${row.id}\n`;
  sql += `INSERT INTO analyses (${cols.join(', ')})\n`;
  sql += `VALUES (${vals.join(', ')})\n`;
  sql += `ON CONFLICT (id) DO UPDATE SET\n`;
  sql += cols.slice(1).map(c => `  ${c} = EXCLUDED.${c}`).join(',\n') + ';\n\n';
});

sql += `-- Re-enable FK checks
SET session_replication_role = 'origin';

-- Verify
SELECT COUNT(*) as inserted_rows FROM analyses;
`;

fs.writeFileSync(OUTPUT_FILE, sql);
console.log(`Generated: ${OUTPUT_FILE}`);
console.log(`Size: ${(sql.length / 1024).toFixed(1)} KB`);
console.log(`Rows: ${data.length}`);
