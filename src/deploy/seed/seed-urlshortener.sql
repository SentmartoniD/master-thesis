CREATE EXTENSION IF NOT EXISTS pg_prewarm;

INSERT INTO links (code, target_url)
SELECT substr(md5(i::text), 1, 12),
       'https://example.com/page/' || i
FROM generate_series(1, 500000) AS i
WHERE NOT EXISTS (SELECT 1 FROM links);

VACUUM ANALYZE links;

SELECT count(*) AS seeded_rows FROM links;