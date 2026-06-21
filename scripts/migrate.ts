import { readFile, readdir } from 'node:fs/promises';
import { resolve } from 'node:path';
import pg from 'pg';

const databaseUrl = process.env.DATABASE_URL;
if (!databaseUrl) throw new Error('DATABASE_URL is required');

const pool = new pg.Pool({ connectionString: databaseUrl });
const directory = resolve('migrations');
const files = (await readdir(directory)).filter((name) => name.endsWith('.sql')).sort();

try {
  for (const file of files) {
    const sql = await readFile(resolve(directory, file), 'utf8');
    console.log(`Applying ${file}`);
    await pool.query(sql);
  }
  console.log('Migrations complete');
} finally {
  await pool.end();
}
