#!/usr/bin/env python3
"""
Amiglot Seed Users — Create all seed users defined in the E2E test plan.

Run this after setting up the test environment (DB + API running).
Usage: python3 scripts/seed-users.py [--api-url http://localhost:6176/api/v1]

Requires: requests (pip install requests)
"""
import argparse
import sys

try:
    import requests
except ImportError:
    print("ERROR: 'requests' package required. Install with: pip install requests", file=sys.stderr)
    sys.exit(1)

DEFAULT_API = "http://localhost:6176/api/v1"

# ---------------------------------------------------------------------------
# Seed user definitions (must match designs/003-end-to-end-test-plan.md §2.2)
# ---------------------------------------------------------------------------
SEED_USERS = [
    {
        "email": "test+seed1@gnailuy.com", "handle": "alice", "tz": "Etc/UTC",
        "langs": [
            {"language_code": "en", "level": 5, "is_native": True, "is_target": False},
            {"language_code": "zh", "level": 2, "is_native": False, "is_target": True},
        ],
        "avail": [{"weekday": d, "start_local_time": "08:00", "end_local_time": "20:00", "timezone": "Etc/UTC"} for d in range(1, 6)],
    },
    {
        "email": "test+seed2@gnailuy.com", "handle": "bob", "tz": "Asia/Shanghai",
        "langs": [
            {"language_code": "zh", "level": 5, "is_native": True, "is_target": False},
            {"language_code": "en", "level": 3, "is_native": False, "is_target": True},
        ],
        "avail": [{"weekday": d, "start_local_time": "16:00", "end_local_time": "23:59", "timezone": "Asia/Shanghai"} for d in range(1, 6)],
    },
    {
        "email": "test+seed3@gnailuy.com", "handle": "carlos", "tz": "America/Sao_Paulo",
        "langs": [
            {"language_code": "pt-BR", "level": 5, "is_native": True, "is_target": False},
            {"language_code": "es", "level": 5, "is_native": True, "is_target": False},
            {"language_code": "en", "level": 3, "is_native": False, "is_target": True},
            {"language_code": "zh", "level": 1, "is_native": False, "is_target": True},
        ],
        "avail": [{"weekday": d, "start_local_time": "09:00", "end_local_time": "18:00", "timezone": "America/Sao_Paulo"} for d in [1, 3, 5]],
    },
    {
        "email": "test+seed4@gnailuy.com", "handle": "diana", "tz": "Etc/UTC",
        "langs": [
            {"language_code": "en", "level": 5, "is_native": True, "is_target": False},
            {"language_code": "pt", "level": 2, "is_native": False, "is_target": True},
        ],
        "avail": [{"weekday": d, "start_local_time": "02:00", "end_local_time": "04:00", "timezone": "Etc/UTC"} for d in [6, 0]],
    },
    {
        "email": "test+seed5@gnailuy.com", "handle": "eve", "tz": "Etc/UTC",
        "langs": [
            {"language_code": "zh", "level": 5, "is_native": True, "is_target": False},
            {"language_code": "en", "level": 3, "is_native": False, "is_target": True},
        ],
        "avail": [{"weekday": d, "start_local_time": "01:00", "end_local_time": "03:00", "timezone": "Etc/UTC"} for d in [6, 0]],
    },
    {
        "email": "test+seed6@gnailuy.com", "handle": "frank", "tz": "Etc/UTC",
        "langs": [
            {"language_code": "en", "level": 5, "is_native": True, "is_target": False},
            {"language_code": "zh", "level": 1, "is_native": False, "is_target": True},
        ],
        "avail": [{"weekday": d, "start_local_time": "07:00", "end_local_time": "09:05", "timezone": "Etc/UTC"} for d in [1, 3]],
    },
    {
        "email": "test+seed7@gnailuy.com", "handle": "grace", "tz": "Asia/Shanghai",
        "langs": [
            {"language_code": "zh-Hans", "level": 5, "is_native": True, "is_target": False},
            {"language_code": "en", "level": 3, "is_native": False, "is_target": True},
        ],
        "avail": [{"weekday": d, "start_local_time": "16:00", "end_local_time": "22:00", "timezone": "Asia/Shanghai"} for d in range(1, 6)],
    },
    {
        "email": "test+seed8@gnailuy.com", "handle": "hiro", "tz": "Asia/Tokyo",
        "langs": [
            {"language_code": "ja", "level": 5, "is_native": True, "is_target": False},
            {"language_code": "ko", "level": 1, "is_native": False, "is_target": True},
        ],
        "avail": [{"weekday": d, "start_local_time": "10:00", "end_local_time": "18:00", "timezone": "Asia/Tokyo"} for d in [1, 3]],
    },
    {
        "email": "test+seed9@gnailuy.com", "handle": "ivan", "tz": "Etc/UTC",
        "langs": [
            {"language_code": "en", "level": 5, "is_native": True, "is_target": False},
            {"language_code": "zh", "level": 2, "is_native": False, "is_target": True},
        ],
        "avail": [{"weekday": d, "start_local_time": "08:00", "end_local_time": "20:00", "timezone": "Etc/UTC"} for d in range(1, 4)],
    },
    {
        "email": "test+seed10@gnailuy.com", "handle": "julia", "tz": "Asia/Shanghai",
        "not_discoverable": True,
        "langs": [
            {"language_code": "zh", "level": 5, "is_native": True, "is_target": False},
            {"language_code": "en", "level": 2, "is_native": False, "is_target": True},
        ],
        "avail": [{"weekday": d, "start_local_time": "10:00", "end_local_time": "18:00", "timezone": "Asia/Shanghai"} for d in [1, 2]],
    },
    {
        "email": "test+seed11@gnailuy.com", "handle": "kevin", "tz": "America/New_York",
        "langs": [
            {"language_code": "en", "level": 5, "is_native": True, "is_target": False},
            {"language_code": "zh", "level": 2, "is_native": False, "is_target": True},
            {"language_code": "pt", "level": 1, "is_native": False, "is_target": True},
        ],
        "avail": [{"weekday": d, "start_local_time": "09:00", "end_local_time": "17:00", "timezone": "America/New_York"} for d in range(1, 6)],
    },
    {
        "email": "test+seed12@gnailuy.com", "handle": "luna", "tz": "America/Sao_Paulo",
        "langs": [
            {"language_code": "pt-BR", "level": 5, "is_native": True, "is_target": False},
            {"language_code": "zh-Hans", "level": 4, "is_native": False, "is_target": False},
            {"language_code": "en", "level": 2, "is_native": False, "is_target": True},
        ],
        "avail": [{"weekday": d, "start_local_time": "10:00", "end_local_time": "18:00", "timezone": "America/Sao_Paulo"} for d in range(1, 6)],
    },
]

# Block relationships: (blocker_handle, blocked_handle)
BLOCKS = [("bob", "ivan")]


def register(api: str, email: str) -> str:
    """Register a user via magic-link flow, return user ID."""
    r = requests.post(f"{api}/auth/magic-link", json={"email": email})
    r.raise_for_status()
    token = r.json()["dev_login_url"].split("token=")[1]
    r2 = requests.post(f"{api}/auth/verify", json={"token": token})
    r2.raise_for_status()
    return r2.json()["user"]["id"]


def put(api: str, path: str, uid: str, body: dict) -> requests.Response:
    r = requests.put(f"{api}{path}", json=body, headers={"X-User-Id": uid})
    if not r.ok:
        print(f"  WARN {path}: {r.status_code} {r.text[:200]}", file=sys.stderr)
    return r


def main():
    parser = argparse.ArgumentParser(description="Create Amiglot seed users")
    parser.add_argument("--api-url", default=DEFAULT_API, help=f"API base URL (default: {DEFAULT_API})")
    parser.add_argument("--db-dsn", default=None, help="PostgreSQL DSN for direct DB operations (blocks, discoverable). If not provided, skips DB-only setup.")
    args = parser.parse_args()
    api = args.api_url

    # Check API is reachable
    try:
        r = requests.get(f"{api}/healthz", timeout=5)
        r.raise_for_status()
    except Exception as e:
        print(f"ERROR: Cannot reach API at {api}: {e}", file=sys.stderr)
        sys.exit(1)

    uid_map: dict[str, str] = {}
    errors = 0

    print("=== Creating seed users ===")
    for i, u in enumerate(SEED_USERS, 1):
        handle = u["handle"]
        print(f"{i:2d}. {handle}...", end=" ", flush=True)
        try:
            uid = register(api, u["email"])
            put(api, "/profile", uid, {"handle": handle, "timezone": u["tz"]})
            put(api, "/profile/languages", uid, {"languages": u["langs"]})
            put(api, "/profile/availability", uid, {"availability": u["avail"]})
            uid_map[handle] = uid
            print(f"OK ({uid})")
        except Exception as e:
            print(f"ERROR: {e}")
            errors += 1

    # DB-only operations (blocks, discoverable overrides)
    conn = None
    if args.db_dsn:
        try:
            import psycopg2
            conn = psycopg2.connect(args.db_dsn)
            conn.autocommit = True
        except ImportError:
            print("WARN: psycopg2 not installed, skipping DB operations", file=sys.stderr)
        except Exception as e:
            print(f"WARN: Cannot connect to DB: {e}", file=sys.stderr)

    # Set not_discoverable users
    for u in SEED_USERS:
        if u.get("not_discoverable") and u["handle"] in uid_map:
            if conn:
                with conn.cursor() as cur:
                    cur.execute("UPDATE profiles SET discoverable = false WHERE user_id = %s", (uid_map[u["handle"]],))
                print(f"  {u['handle']}: set discoverable=false (via DB)")
            else:
                print(f"  WARN: {u['handle']} should be not-discoverable but no DB connection. "
                      "Run manually: UPDATE profiles SET discoverable = false WHERE user_id = '<id>';",
                      file=sys.stderr)

    # Set up blocks
    for blocker_handle, blocked_handle in BLOCKS:
        if blocker_handle in uid_map and blocked_handle in uid_map:
            if conn:
                with conn.cursor() as cur:
                    cur.execute(
                        "INSERT INTO user_blocks (blocker_id, blocked_id) VALUES (%s, %s) ON CONFLICT DO NOTHING",
                        (uid_map[blocker_handle], uid_map[blocked_handle]),
                    )
                print(f"  block: {blocker_handle} -> {blocked_handle} (via DB)")
            else:
                print(f"  WARN: {blocker_handle} should block {blocked_handle} but no DB connection. "
                      f"Run manually: INSERT INTO user_blocks (blocker_id, blocked_id) VALUES ('{uid_map[blocker_handle]}', '{uid_map[blocked_handle]}');",
                      file=sys.stderr)

    if conn:
        conn.close()

    # Verify
    print("\n=== Verification ===")
    for handle, uid in uid_map.items():
        r = requests.get(f"{api}/profile", headers={"X-User-Id": uid})
        if r.ok:
            p = r.json()
            disc = p["profile"]["discoverable"]
            nlangs = len(p["languages"])
            navail = len(p["availability"])
            print(f"  {handle:8s}: discoverable={disc!s:5s}  langs={nlangs}  avail={navail}")

    print(f"\nDone. {len(uid_map)} users created, {errors} errors.")
    if errors:
        sys.exit(1)


if __name__ == "__main__":
    main()
