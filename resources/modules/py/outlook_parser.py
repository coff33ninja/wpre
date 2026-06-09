"""
Outlook data detector and PST/OST extractor.
Does NOT touch accounts or credentials — purely file-level detection + copy.
"""

import json
import os
import shutil
from pathlib import Path


def find_pst_files():
    search_paths = [
        os.path.join(os.environ.get("USERPROFILE", ""), "Documents", "Outlook Files"),
        os.environ.get("LOCALAPPDATA", ""),
        os.environ.get("APPDATA", ""),
        os.path.join(os.environ.get("USERPROFILE", ""), "Desktop"),
    ]
    results = []
    seen = set()
    for search_path in search_paths:
        if not os.path.isdir(search_path):
            continue
        for root, dirs, files in os.walk(search_path):
            for f in files:
                ext = f.lower().split(".")[-1] if "." in f else ""
                if ext in ("pst", "ost"):
                    full_path = os.path.join(root, f)
                    if full_path not in seen:
                        seen.add(full_path)
                        try:
                            size = os.path.getsize(full_path)
                        except OSError:
                            size = 0
                        results.append({
                            "path": full_path,
                            "filename": f,
                            "type": ext.upper(),
                            "size_bytes": size,
                        })
    return results


def find_outlook_registry_profiles():
    import winreg
    profiles = []
    reg_paths = [
        r"Software\Microsoft\Office\16.0\Outlook\Profiles",
        r"Software\Microsoft\Office\15.0\Outlook\Profiles",
        r"Software\Microsoft\Office\14.0\Outlook\Profiles",
        r"Software\Microsoft\Windows NT\CurrentVersion\Windows Messaging Subsystem\Profiles",
    ]
    for reg_path in reg_paths:
        try:
            key = winreg.OpenKey(winreg.HKEY_CURRENT_USER, reg_path, 0, winreg.KEY_READ)
            i = 0
            while True:
                try:
                    profile_name = winreg.EnumKey(key, i)
                    profiles.append({"registry_path": reg_path, "profile_name": profile_name})
                    i += 1
                except OSError:
                    break
            winreg.CloseKey(key)
        except FileNotFoundError:
            continue
    return profiles


def copy_pst_files(pst_list, output_dir):
    results = []
    out_base = os.path.join(output_dir, "Outlook")
    os.makedirs(out_base, exist_ok=True)
    for pst in pst_list:
        dest = os.path.join(out_base, pst["filename"])
        try:
            shutil.copy2(pst["path"], dest)
            results.append({
                "source": pst["path"],
                "destination": dest,
                "success": True,
            })
        except (shutil.Error, OSError) as e:
            results.append({
                "source": pst["path"],
                "destination": dest,
                "success": False,
                "error": str(e),
            })
    return results


def extract_all(output_dir, copy_files=True):
    pst_files = find_pst_files()
    registry_profiles = find_outlook_registry_profiles()
    result = {
        "pst_files_detected": pst_files,
        "registry_profiles": registry_profiles,
        "total_pst_files": len(pst_files),
        "total_registry_profiles": len(registry_profiles),
    }
    if copy_files and pst_files:
        result["copy_results"] = copy_pst_files(pst_files, output_dir)
    return {"success": True, "results": result}


if __name__ == "__main__":
    import sys
    output_dir = sys.argv[1] if len(sys.argv) > 1 else "MigrationVault"
    result = extract_all(output_dir)
    print(json.dumps(result, indent=2, default=str))
