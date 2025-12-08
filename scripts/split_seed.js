/**
 * Split seed into smaller files for Supabase SQL Editor
 * With FK checks disabled
 */

const fs = require('fs');
const path = require('path');

const JSON_FILE = 'C:\\Users\\pradord\\Downloads\\analyses_rows.json';
const OUTPUT_DIR = 'C:\\Users\\pradord\\Documents\\Projects\\imensiah_new\\backend_v3\\migrations\\seed_batches';

// Create output directory
if (!fs.existsSync(OUTPUT_DIR)) {
  fs.mkdirSync(OUTPUT_DIR, { recursive: true });
}

const data = JSON.parse(fs.readFileSync(JSON_FILE, 'utf8'));
const BATCH_SIZE = 3; // 3 rows per file = 12 batches for 36 rows

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

// Process in batches
for (let batchNum = 0; batchNum * BATCH_SIZE < data.length; batchNum++) {
  const start = batchNum * BATCH_SIZE;
  const end = Math.min(start + BATCH_SIZE, data.length);
  const batch = data.slice(start, end);

  let sql = `-- Batch ${batchNum + 1}: rows ${start + 1}-${end} of ${data.length}\n`;
  sql += `-- Run this in Supabase SQL Editor\n`;
  sql += `-- FK checks disabled - submission_id, enrichment_id, company_id set to NULL\n\n`;
  sql += `-- Disable FK checks\n`;
  sql += `SET session_replication_role = 'replica';\n\n`;

  batch.forEach((row, idx) => {
    const cols = [];
    const vals = [];

    cols.push('id'); vals.push(sqlEscape(row.id));
    // Set FK columns to NULL since related data doesn't exist
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

    sql += `-- Row ${start + idx + 1}: ${row.id}\n`;
    sql += `INSERT INTO analyses (${cols.join(', ')})\n`;
    sql += `VALUES (${vals.join(', ')})\n`;
    sql += `ON CONFLICT (id) DO UPDATE SET\n`;
    sql += cols.slice(1).map(c => `  ${c} = EXCLUDED.${c}`).join(',\n') + ';\n\n';
  });

  sql += `-- Re-enable FK checks\n`;
  sql += `SET session_replication_role = 'origin';\n`;

  const filename = path.join(OUTPUT_DIR, `batch_${String(batchNum + 1).padStart(2, '0')}.sql`);
  fs.writeFileSync(filename, sql);
  console.log(`Created: ${filename} (${batch.length} rows)`);
}

console.log(`\nDone! Created ${Math.ceil(data.length / BATCH_SIZE)} batch files in ${OUTPUT_DIR}`);
