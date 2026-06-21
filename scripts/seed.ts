import pg from 'pg';

const databaseUrl = process.env.DATABASE_URL;
if (!databaseUrl) throw new Error('DATABASE_URL is required');

const pool = new pg.Pool({ connectionString: databaseUrl });

try {
  await pool.query(`
    INSERT INTO applications (
      email, email_normalized, username, username_normalized, review_text,
      requested_services, status, email_verified_at, captcha_provider, captcha_verified_at
    ) VALUES
      ('alice@example.com', 'alice@example.com', 'alice', 'alice',
       '我希望加入社区，与朋友使用 Matrix，并发布一些开源项目动态。',
       ARRAY['Matrix', 'GoToSocial'], 'pending', now(), 'turnstile', now()),
      ('lin@example.net', 'lin@example.net', 'lin-dev', 'lin-dev',
       '维护几个开源项目，希望使用 Forgejo，同时参与联邦社交网络。',
       ARRAY['Matrix', 'GoToSocial', 'Forgejo'], 'pending', now(), 'turnstile', now())
    ON CONFLICT DO NOTHING
  `);
  console.log('Seed data inserted');
} finally {
  await pool.end();
}
