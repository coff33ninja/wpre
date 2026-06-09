"""
Firefox browser data extractor.
Reads profiles.ini to find profiles, extracts bookmarks from places.sqlite.
"""

import json
import os
import shutil
import sqlite3
import tempfile
from configparser import ConfigParser


def find_firefox_profiles(profiles_root=None):
    if profiles_root is None:
        profiles_root = os.path.join(
            os.environ.get("APPDATA", ""),
            "Mozilla", "Firefox", "Profiles"
        )
    if not os.path.isdir(profiles_root):
        return []

    profiles_ini = os.path.join(os.path.dirname(profiles_root), "profiles.ini")
    profiles = []

    if os.path.isfile(profiles_ini):
        config = ConfigParser()
        config.read(profiles_ini)
        for section in config.sections():
            if section.startswith("Profile"):
                profile_path = config.get(section, "Path", fallback="")
                is_relative = config.getboolean(section, "IsRelative", fallback=True)
                if is_relative:
                    profile_path = os.path.join(os.path.dirname(profiles_ini), profile_path)
                name = config.get(section, "Name", fallback="")
                if os.path.isdir(profile_path):
                    profiles.append({"name": name, "path": profile_path})
    else:
        for entry in os.listdir(profiles_root):
            full_path = os.path.join(profiles_root, entry)
            if os.path.isdir(full_path):
                profiles.append({"name": entry, "path": full_path})
    return profiles


def extract_bookmarks(profile_path, output_dir, profile_name):
    places_db = os.path.join(profile_path, "places.sqlite")
    if not os.path.isfile(places_db):
        return {"success": False, "error": "places.sqlite not found"}

    tmp_db = tempfile.NamedTemporaryFile(delete=False, suffix=".db")
    tmp_db.close()
    shutil.copy2(places_db, tmp_db.name)

    bookmarks = []
    try:
        conn = sqlite3.connect(tmp_db.name)
        conn.text_factory = str
        cursor = conn.cursor()
        cursor.execute("""
            SELECT b.title, p.url, b.dateAdded, b.parent
            FROM moz_bookmarks b
            JOIN moz_places p ON b.fk = p.id
            WHERE b.type = 1
            ORDER BY b.parent, b.position
        """)
        rows = cursor.fetchall()
        for row in rows:
            bookmarks.append({
                "title": row[0] or "",
                "url": row[1] or "",
                "date_added": row[2],
                "parent": row[3] if row[3] else None,
            })
        conn.close()
    finally:
        os.unlink(tmp_db.name)

    out_dir = os.path.join(output_dir, "Firefox", profile_name)
    os.makedirs(out_dir, exist_ok=True)
    out_path = os.path.join(out_dir, "bookmarks.json")
    with open(out_path, "w", encoding="utf-8") as f:
        json.dump({"bookmarks": bookmarks, "count": len(bookmarks)}, f, indent=2)

    return {"success": True, "entries": len(bookmarks), "path": out_path}


def extract_all(output_dir):
    profiles = find_firefox_profiles()
    if not profiles:
        return {"success": False, "error": "No Firefox profiles found"}
    results = []
    for profile in profiles:
        result = extract_bookmarks(profile["path"], output_dir, profile["name"])
        result["profile_name"] = profile["name"]
        results.append(result)
    return {"success": True, "profiles": results}


if __name__ == "__main__":
    import sys
    output_dir = sys.argv[1] if len(sys.argv) > 1 else "MigrationVault/Browsers"
    result = extract_all(output_dir)
    print(json.dumps(result, indent=2, default=str))
