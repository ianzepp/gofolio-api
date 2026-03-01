-- Seed data for a fresh gofolio-api instance.
-- All INSERTs use ON CONFLICT DO NOTHING so this file is idempotent.

-- Admin user with a known access token.
-- Plaintext token: "gofolio-admin-token"
-- Hash: HMAC-SHA512(key="1234", msg="gofolio-admin-token")
-- To authenticate: POST /api/v1/auth/anonymous {"accessToken":"gofolio-admin-token"}
-- Requires ACCESS_TOKEN_SALT=1234 in the environment.
INSERT INTO "User" (id, "accessToken", provider, role, "createdAt", "updatedAt")
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'f7e63ac60e515fee19eba783fca1ca0cfc6953827fa41b18aff453391b8515789c5059150a8a1f732f83bfd75a74f6eceb46230b119195450bdfab940d52e09c',
    'ANONYMOUS',
    'ADMIN',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- Minimal Property rows so GET /api/v1/info returns valid data.
INSERT INTO "Property" (key, value) VALUES ('BENCHMARKS', '[]') ON CONFLICT (key) DO NOTHING;
