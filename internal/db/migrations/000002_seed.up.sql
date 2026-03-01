-- Seed data for a fresh gofolio-api instance.
-- All INSERTs use ON CONFLICT DO NOTHING so this file is idempotent.

-- Minimal Property rows so GET /api/v1/info returns valid data.
INSERT INTO "Property" (key, value) VALUES ('BENCHMARKS', '[]') ON CONFLICT (key) DO NOTHING;
