"""
Generic browser history SQLite parser (read-only).
Supports Chrome, Edge, and Firefox history extraction.
"""

import json
import os
import sqlite3
import shutil
import tempfile


def parse_chrome_edge_history(db_path, limit=5000):
    tmp_db = tempfile.NamedTemporaryFile(delete=False, suffix=".db")
    tmp_db.close()
    shutil.copy2(db_path, tmp_db.name)
    entries = []
    try:
        conn = sqlite3.connect(tmp_db.name)
        conn.text_factory = str
        cursor = conn.cursor()
        cursor.execute("""
            SELECT url, title, visit_count, last_visit_time
            FROM urls
            ORDER BY last_visit_time DESC
            LIMIT ?
        """, (limit,))
        for row in cursor.fetchall():
            entries.append({
                "url": row[0],
                "title": row[1],
                "visit_count": row[2],
                "last_visit_time": row[3],
            })
        conn.close()
    finally:
        os.unlink(tmp_db.name)
    return entries


def parse_firefox_history(db_path, limit=5000):
    tmp_db = tempfile.NamedTemporaryFile(delete=False, suffix=".db")
    tmp_db.close()
    shutil.copy2(db_path, tmp_db.name)
    entries = []
    try:
        conn = sqlite3.connect(tmp_db.name)
        conn.text_factory = str
        cursor = conn.cursor()
        cursor.execute("""
            SELECT p.url, p.title, p.visit_count, h.visit_date
            FROM moz_historyvisits h
            JOIN moz_places p ON h.place_id = p.id
            ORDER BY h.visit_date DESC
            LIMIT ?
        """, (limit,))
        for row in cursor.fetchall():
            entries.append({
                "url": row[0],
                "title": row[1] or "",
                "visit_count": row[2],
                "visit_date": row[3],
            })
        conn.close()
    finally:
        os.unlink(tmp_db.name)
    return entries


def find_history_dbs():
    dbs = []
    chrome_history = os.path.join(
        os.environ.get("LOCALAPPDATA", ""),
        "Google", "Chrome", "User Data", "Default", "History"
    )
    if os.path.isfile(chrome_history):
        dbs.append({"path": chrome_history, "browser": "Chrome"})
    edge_history = os.path.join(
        os.environ.get("LOCALAPPDATA", ""),
        "Microsoft", "Edge", "User Data", "Default", "History"
    )
    if os.path.isfile(edge_history):
        dbs.append({"path": edge_history, "browser": "Edge"})
    firefox_profiles = os.path.join(os.environ.get("APPDATA", ""), "Mozilla", "Firefox", "Profiles")
    if os.path.isdir(firefox_profiles):
        for profile in os.listdir(firefox_profiles):
            places_db = os.path.join(firefox_profiles, profile, "places.sqlite")
            if os.path.isfile(places_db):
                dbs.append({"path": places_db, "browser": "Firefox", "profile": profile})
    return dbs


if __name__ == "__main__":
    import sys
    output_dir = sys.argv[1] if len(sys.argv) > 1 else "MigrationVault"
    all_results = {}
    dbs = find_history_dbs()
    for db in dbs:
        browser = db["browser"]
        if browser in ("Chrome", "Edge"):
            entries = parse_chrome_edge_history(db["path"])
        elif browser == "Firefox":
            entries = parse_firefox_history(db["path"])
        else:
            continue
        all_results[browser] = {"entries": len(entries), "data": entries}
    result_path = os.path.join(output_dir, "browser_history.json")
    os.makedirs(output_dir, exist_ok=True)
    with open(result_path, "w", encoding="utf-8") as f:
        json.dump(all_results, f, indent=2, default=str)
    print(json.dumps({k: v["entries"] for k, v in all_results.items()}, indent=2))
