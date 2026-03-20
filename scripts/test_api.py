#!/usr/bin/env python3
"""Quick smoke test for Logfire Query API — GET /v1/query?sql=..."""

import os
import sys

try:
    import requests
except ImportError:
    sys.exit("pip install requests")

TOKEN = os.environ.get("LOGFIRE_READ_TOKEN", "")
BASE_URL = os.environ.get("LOGFIRE_BASE_URL", "https://logfire-us.pydantic.dev")
TRACE_ID = sys.argv[1] if len(sys.argv) > 1 else "019d0bb48e7a2598220e44a22eafc8a1"

if not TOKEN:
    sys.exit("Set LOGFIRE_READ_TOKEN env var")

sql = f"SELECT * FROM records WHERE trace_id = '{TRACE_ID}' ORDER BY start_timestamp"
url = f"{BASE_URL}/v1/query"

resp = requests.get(url, params={"sql": sql}, headers={"Authorization": f"Bearer {TOKEN}"})
print(f"Status: {resp.status_code}")
print(resp.text[:2000] if len(resp.text) > 2000 else resp.text)
