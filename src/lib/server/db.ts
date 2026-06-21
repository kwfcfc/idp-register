import pg from 'pg';
import { config } from './env';

const { Pool } = pg;

const globalForDb = globalThis as unknown as { inviteAdminPool?: pg.Pool };

export const pool =
  globalForDb.inviteAdminPool ??
  new Pool({
    connectionString: config.databaseUrl,
    max: 10,
    idleTimeoutMillis: 30_000,
    connectionTimeoutMillis: 5_000
  });

if (process.env.NODE_ENV !== 'production') globalForDb.inviteAdminPool = pool;

export async function query<T extends pg.QueryResultRow = pg.QueryResultRow>(
  text: string,
  params: unknown[] = []
): Promise<pg.QueryResult<T>> {
  return pool.query<T>(text, params);
}
