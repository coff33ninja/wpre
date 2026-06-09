"""
File deduplication engine.
Indexes files by SHA256 hash, identifies duplicates, reports savings.
"""

import hashlib
import json
import os
from pathlib import Path


def hash_file(path, blocksize=65536):
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for block in iter(lambda: f.read(blocksize), b""):
            h.update(block)
    return h.hexdigest()


def index_files(root_dir, exclude_patterns=None):
    exclude = exclude_patterns or []
    index = {}
    for root, dirs, files in os.walk(root_dir):
        for f in files:
            path = os.path.join(root, f)
            rel_path = os.path.relpath(path, root_dir)
            skipped = False
            for pat in exclude:
                if f.endswith(pat.replace("*", "")):
                    skipped = True
                    break
            if skipped:
                continue
            try:
                file_hash = hash_file(path)
                size = os.path.getsize(path)
            except (OSError, PermissionError):
                continue
            if file_hash not in index:
                index[file_hash] = {"hash": file_hash, "size": size, "paths": []}
            index[file_hash]["paths"].append(rel_path)
    return index


def find_duplicates(index):
    return {h: e for h, e in index.items() if len(e["paths"]) > 1}


def calculate_savings(duplicates):
    total_wasted = 0
    for entry in duplicates.values():
        total_wasted += entry["size"] * (len(entry["paths"]) - 1)
    return total_wasted


def deduplicate_report(root_dir, output_path, exclude_patterns=None):
    index = index_files(root_dir, exclude_patterns)
    duplicates = find_duplicates(index)
    savings = calculate_savings(duplicates)
    report = {
        "root_directory": root_dir,
        "total_files": len(index),
        "unique_files": len(index) - len(duplicates),
        "duplicate_groups": len(duplicates),
        "wasted_bytes": savings,
        "wasted_human": f"{savings / (1024**3):.2f} GB" if savings > 1024**3
            else f"{savings / (1024**2):.2f} MB" if savings > 1024**2
            else f"{savings / 1024:.2f} KB",
        "duplicates": {h: e for h, e in duplicates.items()},
    }
    with open(output_path, "w") as f:
        json.dump(report, f, indent=2)
    return report


if __name__ == "__main__":
    import sys
    root = sys.argv[1] if len(sys.argv) > 1 else "."
    output = sys.argv[2] if len(sys.argv) > 2 else "dedup_report.json"
    report = deduplicate_report(root, output)
    print(json.dumps({
        "total_files": report["total_files"],
        "unique_files": report["unique_files"],
        "duplicate_groups": report["duplicate_groups"],
        "wasted": report["wasted_human"],
    }, indent=2))
