"""
Application detector for key user apps.
Scans for installed/config files for outlook, browsers, dev tools, etc.
"""

import json
import os
import subprocess


def check_registry_installed(app_name, reg_path, value_name="DisplayName"):
    try:
        result = subprocess.run(
            ["reg", "query", reg_path, "/v", value_name],
            capture_output=True, text=True, timeout=10
        )
        return result.returncode == 0
    except (subprocess.TimeoutExpired, FileNotFoundError):
        return False


def detect_chrome():
    paths = [
        os.path.join(os.environ.get("LOCALAPPDATA", ""), "Google", "Chrome", "Application", "chrome.exe"),
        os.path.join(os.environ.get("PROGRAMFILES(X86)", ""), "Google", "Chrome", "Application", "chrome.exe"),
    ]
    for p in paths:
        if os.path.isfile(p):
            return {"installed": True, "path": p, "version": _get_version(p)}
    return {"installed": False}


def detect_edge():
    paths = [
        os.path.join(os.environ.get("LOCALAPPDATA", ""), "Microsoft", "Edge", "Application", "msedge.exe"),
        os.path.join(os.environ.get("PROGRAMFILES(X86)", ""), "Microsoft", "Edge", "Application", "msedge.exe"),
    ]
    for p in paths:
        if os.path.isfile(p):
            return {"installed": True, "path": p, "version": _get_version(p)}
    return {"installed": False}


def detect_firefox():
    paths = [
        os.path.join(os.environ.get("PROGRAMFILES", ""), "Mozilla Firefox", "firefox.exe"),
        os.path.join(os.environ.get("PROGRAMFILES(X86)", ""), "Mozilla Firefox", "firefox.exe"),
    ]
    for p in paths:
        if os.path.isfile(p):
            return {"installed": True, "path": p, "version": _get_version(p)}
    return {"installed": False}


def detect_outlook():
    paths = [
        os.path.join(os.environ.get("PROGRAMFILES", ""), "Microsoft Office", "root", "Office16", "OUTLOOK.EXE"),
        os.path.join(os.environ.get("PROGRAMFILES(X86)", ""), "Microsoft Office", "root", "Office16", "OUTLOOK.EXE"),
    ]
    for p in paths:
        if os.path.isfile(p):
            return {"installed": True, "path": p}
    return {"installed": False}


def detect_vscode():
    paths = [
        os.path.join(os.environ.get("LOCALAPPDATA", ""), "Programs", "Microsoft VS Code", "Code.exe"),
        os.path.join(os.environ.get("PROGRAMFILES", ""), "Microsoft VS Code", "Code.exe"),
    ]
    for p in paths:
        if os.path.isfile(p):
            return {"installed": True, "path": p}
    return {"installed": False}


def detect_onedrive():
    paths = [
        os.path.join(os.environ.get("LOCALAPPDATA", ""), "Microsoft", "OneDrive", "OneDrive.exe"),
        os.path.join(os.environ.get("PROGRAMFILES", ""), "Microsoft OneDrive", "OneDrive.exe"),
    ]
    for p in paths:
        if os.path.isfile(p):
            return {"installed": True, "path": p}
    return {"installed": False}


def _get_version(exe_path):
    try:
        result = subprocess.run(
            ["powershell", "-Command", f"(Get-Item '{exe_path}').VersionInfo.FileVersion"],
            capture_output=True, text=True, timeout=5
        )
        return result.stdout.strip()
    except (subprocess.TimeoutExpired, FileNotFoundError):
        return None


def detect_all():
    return {
        "chrome": detect_chrome(),
        "edge": detect_edge(),
        "firefox": detect_firefox(),
        "outlook": detect_outlook(),
        "vscode": detect_vscode(),
        "onedrive": detect_onedrive(),
    }


if __name__ == "__main__":
    result = detect_all()
    print(json.dumps(result, indent=2))
