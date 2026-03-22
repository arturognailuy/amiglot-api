-- seed_test_profiles.sql
-- Prefills test profiles for discovery & matching E2E scenarios.
-- Run after DB redeploy: psql -f db/seeds/seed_test_profiles.sql
--
-- All test users use emails: test+seedN@gnailuy.com
-- All passwords/auth handled via dev magic links (no tokens seeded).
--
-- Scenario coverage:
--   1. Basic mutual match (English ↔ Chinese)
--   2. Multi-language match (teaches 2+ languages)
--   3. Bridge-only match (bridge but different teach/learn combos)
--   4. No overlap (language match but zero availability overlap)
--   5. Minimal overlap (just above 60-min threshold)
--   6. Multiple timezone match (availability in different TZs still overlaps)
--   7. One-way mismatch (supply but no demand — should NOT match)
--   8. Rare language target (no matches expected)
--   9. Blocked user pair
--  10. Non-discoverable user (should NOT appear)
--  11. Base-language matching (zh vs zh-Hans)
--  12. Three-way language exchange (pt ↔ en ↔ zh)

BEGIN;

-- Clean previous seed data (idempotent)
DELETE FROM user_blocks WHERE blocker_id IN (SELECT id FROM users WHERE email LIKE 'test+seed%@gnailuy.com');
DELETE FROM availability_slots WHERE user_id IN (SELECT id FROM users WHERE email LIKE 'test+seed%@gnailuy.com');
DELETE FROM user_languages WHERE user_id IN (SELECT id FROM users WHERE email LIKE 'test+seed%@gnailuy.com');
DELETE FROM profiles WHERE user_id IN (SELECT id FROM users WHERE email LIKE 'test+seed%@gnailuy.com');
DELETE FROM users WHERE email LIKE 'test+seed%@gnailuy.com';

-- ============================================================
-- Users
-- ============================================================

INSERT INTO users (id, email) VALUES
  ('a0000001-0000-0000-0000-000000000001', 'test+seed1@gnailuy.com'),   -- Alice: English native, targets Chinese
  ('a0000001-0000-0000-0000-000000000002', 'test+seed2@gnailuy.com'),   -- Bob: Chinese native, targets English
  ('a0000001-0000-0000-0000-000000000003', 'test+seed3@gnailuy.com'),   -- Carlos: Portuguese + Spanish native, targets English + Chinese
  ('a0000001-0000-0000-0000-000000000004', 'test+seed4@gnailuy.com'),   -- Diana: English native, targets Portuguese (no Chinese)
  ('a0000001-0000-0000-0000-000000000005', 'test+seed5@gnailuy.com'),   -- Eve: Chinese native, targets English (no overlap hours)
  ('a0000001-0000-0000-0000-000000000006', 'test+seed6@gnailuy.com'),   -- Frank: English native, targets Chinese (minimal overlap)
  ('a0000001-0000-0000-0000-000000000007', 'test+seed7@gnailuy.com'),   -- Grace: zh-Hans native, targets en (base-language test)
  ('a0000001-0000-0000-0000-000000000008', 'test+seed8@gnailuy.com'),   -- Hiro: Japanese native, targets Korean (rare — no matches)
  ('a0000001-0000-0000-0000-000000000009', 'test+seed9@gnailuy.com'),   -- Ivan: English native, targets Chinese (blocked by Bob)
  ('a0000001-0000-0000-0000-000000000010', 'test+seed10@gnailuy.com'),  -- Julia: Chinese native, targets English (NOT discoverable)
  ('a0000001-0000-0000-0000-000000000011', 'test+seed11@gnailuy.com'),  -- Kevin: English native, targets zh + pt (multi-language match)
  ('a0000001-0000-0000-0000-000000000012', 'test+seed12@gnailuy.com');  -- Luna: pt-BR native + zh-Hans advanced, targets en

-- ============================================================
-- Profiles
-- ============================================================

INSERT INTO profiles (user_id, handle, handle_norm, birth_year, birth_month, country_code, timezone, discoverable) VALUES
  ('a0000001-0000-0000-0000-000000000001', 'alice',   'alice',   1995, 3,  'US', 'America/New_York',      true),
  ('a0000001-0000-0000-0000-000000000002', 'bob',     'bob',     1992, 7,  'CN', 'Asia/Shanghai',         true),
  ('a0000001-0000-0000-0000-000000000003', 'carlos',  'carlos',  1990, 1,  'BR', 'America/Sao_Paulo',     true),
  ('a0000001-0000-0000-0000-000000000004', 'diana',   'diana',   1998, 11, 'GB', 'Europe/London',         true),
  ('a0000001-0000-0000-0000-000000000005', 'eve',     'eve',     1996, 5,  'CN', 'Asia/Shanghai',         true),
  ('a0000001-0000-0000-0000-000000000006', 'frank',   'frank',   1993, 9,  'US', 'America/Los_Angeles',   true),
  ('a0000001-0000-0000-0000-000000000007', 'grace',   'grace',   1997, 2,  'CN', 'Asia/Shanghai',         true),
  ('a0000001-0000-0000-0000-000000000008', 'hiro',    'hiro',    1994, 8,  'JP', 'Asia/Tokyo',            true),
  ('a0000001-0000-0000-0000-000000000009', 'ivan',    'ivan',    1991, 4,  'RU', 'Europe/Moscow',         true),
  ('a0000001-0000-0000-0000-000000000010', 'julia',   'julia',   2000, 6,  'CN', 'Asia/Shanghai',         false),  -- NOT discoverable
  ('a0000001-0000-0000-0000-000000000011', 'kevin',   'kevin',   1989, 10, 'CA', 'America/Vancouver',     true),
  ('a0000001-0000-0000-0000-000000000012', 'luna',    'luna',    1999, 12, 'BR', 'America/Sao_Paulo',     true);

-- ============================================================
-- Languages
-- level: 0=zero, 1=beginner, 2=elementary, 3=intermediate, 4=advanced, 5=native
-- ============================================================

INSERT INTO user_languages (user_id, language_code, level, is_native, is_target, sort_order) VALUES
  -- Alice: en native, targets zh
  ('a0000001-0000-0000-0000-000000000001', 'en', 5, true,  false, 0),
  ('a0000001-0000-0000-0000-000000000001', 'zh', 2, false, true,  1),

  -- Bob: zh native, en advanced, targets en
  ('a0000001-0000-0000-0000-000000000002', 'zh',      5, true,  false, 0),
  ('a0000001-0000-0000-0000-000000000002', 'en',      4, false, true,  1),

  -- Carlos: pt-BR native, es native, en intermediate, targets en + zh
  ('a0000001-0000-0000-0000-000000000003', 'pt-BR',   5, true,  false, 0),
  ('a0000001-0000-0000-0000-000000000003', 'es',      5, true,  false, 1),
  ('a0000001-0000-0000-0000-000000000003', 'en',      3, false, true,  2),
  ('a0000001-0000-0000-0000-000000000003', 'zh',      1, false, true,  3),

  -- Diana: en native, pt intermediate, targets pt
  ('a0000001-0000-0000-0000-000000000004', 'en',      5, true,  false, 0),
  ('a0000001-0000-0000-0000-000000000004', 'pt',      3, false, true,  1),

  -- Eve: zh native, en advanced, targets en (availability won't overlap with Alice)
  ('a0000001-0000-0000-0000-000000000005', 'zh',      5, true,  false, 0),
  ('a0000001-0000-0000-0000-000000000005', 'en',      4, false, true,  1),

  -- Frank: en native, zh elementary, targets zh
  ('a0000001-0000-0000-0000-000000000006', 'en',      5, true,  false, 0),
  ('a0000001-0000-0000-0000-000000000006', 'zh',      2, false, true,  1),

  -- Grace: zh-Hans native, en advanced, targets en (base-language test with Alice who targets 'zh')
  ('a0000001-0000-0000-0000-000000000007', 'zh-Hans', 5, true,  false, 0),
  ('a0000001-0000-0000-0000-000000000007', 'en',      4, false, true,  1),

  -- Hiro: ja native, targets ko (rare combo — nobody teaches Korean)
  ('a0000001-0000-0000-0000-000000000008', 'ja',      5, true,  false, 0),
  ('a0000001-0000-0000-0000-000000000008', 'ko',      1, false, true,  1),

  -- Ivan: en native, zh beginner, targets zh
  ('a0000001-0000-0000-0000-000000000009', 'en',      5, true,  false, 0),
  ('a0000001-0000-0000-0000-000000000009', 'zh',      1, false, true,  1),

  -- Julia: zh native, en advanced, targets en (but NOT discoverable)
  ('a0000001-0000-0000-0000-000000000010', 'zh',      5, true,  false, 0),
  ('a0000001-0000-0000-0000-000000000010', 'en',      4, false, true,  1),

  -- Kevin: en native, zh elementary, pt beginner, targets zh + pt
  ('a0000001-0000-0000-0000-000000000011', 'en',      5, true,  false, 0),
  ('a0000001-0000-0000-0000-000000000011', 'zh',      2, false, true,  1),
  ('a0000001-0000-0000-0000-000000000011', 'pt',      1, false, true,  2),

  -- Luna: pt-BR native, zh-Hans advanced, en advanced, targets en
  ('a0000001-0000-0000-0000-000000000012', 'pt-BR',   5, true,  false, 0),
  ('a0000001-0000-0000-0000-000000000012', 'zh-Hans', 4, false, false, 1),
  ('a0000001-0000-0000-0000-000000000012', 'en',      4, false, true,  2);

-- ============================================================
-- Availability slots (weekday: 0=Sun, 1=Mon, ..., 6=Sat)
-- Wide slots for most users; Eve gets non-overlapping times.
-- ============================================================

INSERT INTO availability_slots (user_id, weekday, start_local_time, end_local_time, timezone, sort_order) VALUES
  -- Alice: Mon-Fri 18:00-22:00 ET
  ('a0000001-0000-0000-0000-000000000001', 1, '18:00', '22:00', 'America/New_York', 0),
  ('a0000001-0000-0000-0000-000000000001', 2, '18:00', '22:00', 'America/New_York', 1),
  ('a0000001-0000-0000-0000-000000000001', 3, '18:00', '22:00', 'America/New_York', 2),
  ('a0000001-0000-0000-0000-000000000001', 4, '18:00', '22:00', 'America/New_York', 3),
  ('a0000001-0000-0000-0000-000000000001', 5, '18:00', '22:00', 'America/New_York', 4),

  -- Bob: Mon-Fri 08:00-12:00 CST (= Mon-Fri 00:00-04:00 UTC, overlaps with Alice's 22:00-02:00 UTC)
  ('a0000001-0000-0000-0000-000000000002', 1, '08:00', '12:00', 'Asia/Shanghai', 0),
  ('a0000001-0000-0000-0000-000000000002', 2, '08:00', '12:00', 'Asia/Shanghai', 1),
  ('a0000001-0000-0000-0000-000000000002', 3, '08:00', '12:00', 'Asia/Shanghai', 2),
  ('a0000001-0000-0000-0000-000000000002', 4, '08:00', '12:00', 'Asia/Shanghai', 3),
  ('a0000001-0000-0000-0000-000000000002', 5, '08:00', '12:00', 'Asia/Shanghai', 4),

  -- Carlos: Mon-Fri 19:00-23:00 BRT
  ('a0000001-0000-0000-0000-000000000003', 1, '19:00', '23:00', 'America/Sao_Paulo', 0),
  ('a0000001-0000-0000-0000-000000000003', 2, '19:00', '23:00', 'America/Sao_Paulo', 1),
  ('a0000001-0000-0000-0000-000000000003', 3, '19:00', '23:00', 'America/Sao_Paulo', 2),
  ('a0000001-0000-0000-0000-000000000003', 4, '19:00', '23:00', 'America/Sao_Paulo', 3),
  ('a0000001-0000-0000-0000-000000000003', 5, '19:00', '23:00', 'America/Sao_Paulo', 4),

  -- Diana: Mon-Fri 18:00-22:00 GMT
  ('a0000001-0000-0000-0000-000000000004', 1, '18:00', '22:00', 'Europe/London', 0),
  ('a0000001-0000-0000-0000-000000000004', 2, '18:00', '22:00', 'Europe/London', 1),
  ('a0000001-0000-0000-0000-000000000004', 3, '18:00', '22:00', 'Europe/London', 2),
  ('a0000001-0000-0000-0000-000000000004', 4, '18:00', '22:00', 'Europe/London', 3),
  ('a0000001-0000-0000-0000-000000000004', 5, '18:00', '22:00', 'Europe/London', 4),

  -- Eve: Mon-Fri 01:00-03:00 CST (= 17:00-19:00 UTC — does NOT overlap with Alice's 22:00-02:00 UTC)
  ('a0000001-0000-0000-0000-000000000005', 1, '01:00', '03:00', 'Asia/Shanghai', 0),
  ('a0000001-0000-0000-0000-000000000005', 2, '01:00', '03:00', 'Asia/Shanghai', 1),
  ('a0000001-0000-0000-0000-000000000005', 3, '01:00', '03:00', 'Asia/Shanghai', 2),
  ('a0000001-0000-0000-0000-000000000005', 4, '01:00', '03:00', 'Asia/Shanghai', 3),
  ('a0000001-0000-0000-0000-000000000005', 5, '01:00', '03:00', 'Asia/Shanghai', 4),

  -- Frank: Mon 07:00-08:05 PT (= 15:00-16:05 UTC — exactly 65 min, minimal overlap with Bob's 00:00-04:00)
  -- Actually, let's make Frank overlap with Bob. Bob is 00:00-04:00 UTC. Frank needs to overlap there.
  -- Frank: Mon 16:00-17:05 PT (= Tue 00:00-01:05 UTC — overlaps Bob's Tue 00:00-04:00 by 65 min)
  ('a0000001-0000-0000-0000-000000000006', 2, '00:00', '01:05', 'UTC', 0),

  -- Grace: Mon-Fri 08:00-12:00 CST (same as Bob)
  ('a0000001-0000-0000-0000-000000000007', 1, '08:00', '12:00', 'Asia/Shanghai', 0),
  ('a0000001-0000-0000-0000-000000000007', 2, '08:00', '12:00', 'Asia/Shanghai', 1),
  ('a0000001-0000-0000-0000-000000000007', 3, '08:00', '12:00', 'Asia/Shanghai', 2),
  ('a0000001-0000-0000-0000-000000000007', 4, '08:00', '12:00', 'Asia/Shanghai', 3),
  ('a0000001-0000-0000-0000-000000000007', 5, '08:00', '12:00', 'Asia/Shanghai', 4),

  -- Hiro: Mon-Fri 19:00-22:00 JST
  ('a0000001-0000-0000-0000-000000000008', 1, '19:00', '22:00', 'Asia/Tokyo', 0),
  ('a0000001-0000-0000-0000-000000000008', 2, '19:00', '22:00', 'Asia/Tokyo', 1),
  ('a0000001-0000-0000-0000-000000000008', 3, '19:00', '22:00', 'Asia/Tokyo', 2),

  -- Ivan: Mon-Fri 18:00-22:00 MSK (overlaps with Bob)
  ('a0000001-0000-0000-0000-000000000009', 1, '18:00', '22:00', 'Europe/Moscow', 0),
  ('a0000001-0000-0000-0000-000000000009', 2, '18:00', '22:00', 'Europe/Moscow', 1),
  ('a0000001-0000-0000-0000-000000000009', 3, '18:00', '22:00', 'Europe/Moscow', 2),
  ('a0000001-0000-0000-0000-000000000009', 4, '18:00', '22:00', 'Europe/Moscow', 3),
  ('a0000001-0000-0000-0000-000000000009', 5, '18:00', '22:00', 'Europe/Moscow', 4),

  -- Julia: Mon-Fri 08:00-12:00 CST (same as Bob — but she's NOT discoverable)
  ('a0000001-0000-0000-0000-000000000010', 1, '08:00', '12:00', 'Asia/Shanghai', 0),
  ('a0000001-0000-0000-0000-000000000010', 2, '08:00', '12:00', 'Asia/Shanghai', 1),
  ('a0000001-0000-0000-0000-000000000010', 3, '08:00', '12:00', 'Asia/Shanghai', 2),

  -- Kevin: Mon-Sat 17:00-22:00 PT
  ('a0000001-0000-0000-0000-000000000011', 1, '17:00', '22:00', 'America/Vancouver', 0),
  ('a0000001-0000-0000-0000-000000000011', 2, '17:00', '22:00', 'America/Vancouver', 1),
  ('a0000001-0000-0000-0000-000000000011', 3, '17:00', '22:00', 'America/Vancouver', 2),
  ('a0000001-0000-0000-0000-000000000011', 4, '17:00', '22:00', 'America/Vancouver', 3),
  ('a0000001-0000-0000-0000-000000000011', 5, '17:00', '22:00', 'America/Vancouver', 4),
  ('a0000001-0000-0000-0000-000000000011', 6, '17:00', '22:00', 'America/Vancouver', 5),

  -- Luna: Mon-Fri 19:00-23:00 BRT
  ('a0000001-0000-0000-0000-000000000012', 1, '19:00', '23:00', 'America/Sao_Paulo', 0),
  ('a0000001-0000-0000-0000-000000000012', 2, '19:00', '23:00', 'America/Sao_Paulo', 1),
  ('a0000001-0000-0000-0000-000000000012', 3, '19:00', '23:00', 'America/Sao_Paulo', 2),
  ('a0000001-0000-0000-0000-000000000012', 4, '19:00', '23:00', 'America/Sao_Paulo', 3),
  ('a0000001-0000-0000-0000-000000000012', 5, '19:00', '23:00', 'America/Sao_Paulo', 4);

-- ============================================================
-- Blocks
-- ============================================================

-- Bob blocks Ivan (both directions tested by the query)
INSERT INTO user_blocks (blocker_id, blocked_id) VALUES
  ('a0000001-0000-0000-0000-000000000002', 'a0000001-0000-0000-0000-000000000009');

COMMIT;

-- ============================================================
-- Expected match scenarios (for E2E validation):
--
-- Alice → sees Luna (20h), Bob (8h), Grace (8h);
--         does NOT see Eve (no overlap), Julia (not discoverable)
-- Bob → sees Grace (20h), Kevin (16h), Alice (8h), Luna (8h), Frank (1h);
--        does NOT see Ivan (blocked), Julia (not discoverable)
-- Carlos → sees Kevin (10h, targets pt, bridge en);
--          does NOT see Diana (no time overlap)
-- Diana → sees nobody (18:00-22:00 UTC has zero overlap with Carlos/Luna at 22:00-02:00 UTC)
-- Eve → sees Ivan (8h)
-- Frank → sees Bob (1h), Grace (1h), Luna (1h)
-- Grace → sees Bob (20h), Kevin (16h), Alice (8h), Luna (8h), Frank (1h)
-- Hiro → sees nobody (no Korean teachers)
-- Ivan → sees Eve (8h); does NOT see Bob (blocked)
-- Kevin → sees Bob (16h), Grace (16h), Carlos (10h), Luna (10h)
-- Luna → sees Alice (20h), Kevin (10h), Bob (8h), Grace (8h), Frank (1h)
-- ============================================================
