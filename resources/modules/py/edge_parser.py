"""
Microsoft Edge browser data extractor.
Structure mirrors Chrome — extracts bookmarks, extensions, optional history.
"""

import json
import os
import shutil
import sqlite3
import tempfile
from pathlib import Path


def find_edge_profile(user_data_root=None):
    if user_data_root is None:
        user_data_root = os.path.join(
            os.environ.get("LOCALAPPDATA", ""),
            "Microsoft", "Edge", "User Data"
        )
    default_profile = os.path.join(user_data_root, "Default")
    if os.path.isdir(default_profile):
        return default_profile
    return None


def extract_bookmarks(profile_path, output_dir):
    bookmarks_path = os.path.join(profile_path, "Bookmarks")
    if not os.path.isfile(bookmarks_path):
        return {"success": False, "error": "Bookmarks file not found"}
    out_path = os.path.join(output_dir, "Edge", "Bookmarks.json")
    os.makedirs(os.path.dirname(out_path), exist_ok=True)
    shutil.copy2(bookmarks_path, out_path)
    return {"success": True, "path": out_path}


def extract_extensions(profile_path, output_dir):
    extensions_dir = os.path.join(profile_path, "Extensions")
    if not os.path.isdir(extensions_dir):
        return {"success": False, "error": "Extensions directory not found"}
    out_dir = os.path.join(output_dir, "Edge", "Extensions")
    os.makedirs(out_dir, exist_ok=True)
    ext_list = []
    for ext_id in os.listdir(extensions_dir):
        if os.path.isdir(os.path.join(extensions_dir, ext_id)):
            ext_list.append(ext_id)
    manifest_path = os.path.join(out_dir, "extension_ids.json")
    with open(manifest_path, "w") as f:
        json.dump({"extension_ids": ext_list, "count": len(ext_list)}, f, indent=2)
    return {"success": True, "extensions_found": len(ext_list)}


def extract_history(profile_path, output_dir):
    history_db = os.path.join(profile_path, "History")
    if not os.path.isfile(history_db):
        return {"success": False, "error": "History database not found"}
    tmp_db = tempfile.NamedTemporaryFile(delete=False, suffix=".db")
    tmp_db.close()
    shutil.copy2(history_db, tmp_db.name)
    urls = []
    try:
        conn = sqlite3.connect(tmp_db.name)
        cursor = conn.cursor()
        cursor.execute(
            "SELECT url, title, visit_count, last_visit_time FROM urls ORDER BY last_visit_time DESC LIMIT 1000"
        )
        rows = cursor.fetchall()
        for row in rows:
            urls.append({
                "url": row[0],
                "title": row[1],
                "visit_count": row[2],
                "last_visit_time": row[3],
            })
        conn.close()
    finally:
        os.unlink(tmp_db.name)
    out_path = os.path.join(output_dir, "Edge", "history.json")
    os.makedirs(os.path.dirname(out_path), exist_ok=True)
    with open(out_path, "w", encoding="utf-8") as f:
        json.dump({"urls": urls, "count": len(urls)}, f, indent=2)
    return {"success": True, "entries": len(urls)}


def extract_all(output_dir, include_history=False):
    profile_path = find_edge_profile()
    if not profile_path:
        return {"success": False, "error": "Edge profile not found"}
    results = {
        "profile_path": profile_path,
        "bookmarks": extract_bookmarks(profile_path, output_dir),
        "extensions": extract_extensions(profile_path, output_dir),
    }
    if include_history:
        results["history"] = extract_history(profile_path, output_dir)
    return {"success": True, "results": results}


if __name__ == "__main__":
    import sys
    output_dir = sys.argv[1] if len(sys.argv) > 1 else "MigrationVault/Browsers"
    include_history = "--history" in sys.argv
    result = extract_all(output_dir, include_history)
    print(json.dumps(result, indent=2, default=str))
