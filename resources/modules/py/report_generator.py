"""
Final migration report generator.
Consumes the master manifest and produces JSON and/or HTML reports.
"""

import json
import os
from datetime import datetime


def generate_html_report(manifest, output_path):
    html = f"""<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<title>WPRE Migration Report</title>
<style>
  body {{ font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; margin: 40px; color: #333; }}
  h1 {{ border-bottom: 2px solid #0066cc; padding-bottom: 10px; }}
  h2 {{ color: #0066cc; margin-top: 30px; }}
  table {{ border-collapse: collapse; width: 100%; margin: 15px 0; }}
  th, td {{ text-align: left; padding: 8px 12px; border: 1px solid #ddd; }}
  th {{ background: #0066cc; color: white; }}
  .success {{ color: #28a745; font-weight: bold; }}
  .warning {{ color: #ffc107; font-weight: bold; }}
  .error {{ color: #dc3545; font-weight: bold; }}
  .summary {{ display: flex; gap: 20px; }}
  .stat {{ background: #f8f9fa; border-radius: 8px; padding: 20px; flex: 1; text-align: center; }}
  .stat-value {{ font-size: 28px; font-weight: bold; color: #0066cc; }}
  .stat-label {{ font-size: 12px; color: #666; text-transform: uppercase; }}
</style>
</head>
<body>
<h1>WPRE Migration Report</h1>
<p>Generated: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}</p>
<h2>Summary</h2>
<div class="summary">
  <div class="stat"><div class="stat-value">{manifest.get('data', {}).get('files_total', 0)}</div><div class="stat-label">Total Files</div></div>
  <div class="stat"><div class="stat-value">{manifest.get('data', {}).get('files_copied', 0)}</div><div class="stat-label">Files Copied</div></div>
  <div class="stat"><div class="stat-value">{manifest.get('data', {}).get('files_error', 0)}</div><div class="stat-label">Errors</div></div>
  <div class="stat"><div class="stat-value">{_format_bytes(manifest.get('data', {}).get('size_bytes', 0))}</div><div class="stat-label">Total Size</div></div>
</div>
<h2>Pipeline Stages</h2>
<table>
  <tr><th>Stage</th><th>Status</th></tr>
"""
    for stage in manifest.get("stages_completed", []):
        html += f"  <tr><td>{stage}</td><td class='success'>completed</td></tr>\n"
    html += """</table>
<h2>OneDrive</h2>
<table>
"""
    od = manifest.get("onedrive", {})
    html += f"  <tr><td>Detected</td><td>{od.get('detected', False)}</td></tr>\n"
    html += f"  <tr><td>Harvested</td><td class='{'success' if od.get('harvested') else 'error'}'>{od.get('harvested', False)}</td></tr>\n"
    html += f"  <tr><td>Integrity Verified</td><td class='{'success' if od.get('integrity_verified') else 'warning'}'>{od.get('integrity_verified', False)}</td></tr>\n"
    html += f"  <tr><td>Placeholders Resolved</td><td>{od.get('placeholders_resolved', 0)}</td></tr>\n"
    html += """</table>
<h2>Applications</h2>
<table>
  <tr><th>App</th><th>Status</th></tr>
"""
    for app, captured in manifest.get("apps", {}).items():
        html += f"  <tr><td>{app}</td><td class='{'success' if captured else 'warning'}'>{'captured' if captured else 'skipped'}</td></tr>\n"
    html += """</table>
<h2>Errors</h2>
<table>
  <tr><th>Stage</th><th>File</th><th>Error</th></tr>
"""
    for err in manifest.get("errors", []):
        html += f"  <tr><td>{err.get('stage', '')}</td><td>{err.get('file', '')}</td><td class='error'>{err.get('error', '')}</td></tr>\n"
    html += """
</table>
</body>
</html>"""
    with open(output_path, "w", encoding="utf-8") as f:
        f.write(html)
    return {"success": True, "path": output_path}


def _format_bytes(b):
    if b >= 1024**4:
        return f"{b / 1024**4:.2f} TB"
    elif b >= 1024**3:
        return f"{b / 1024**3:.2f} GB"
    elif b >= 1024**2:
        return f"{b / 1024**2:.2f} MB"
    elif b >= 1024:
        return f"{b / 1024:.2f} KB"
    return f"{b} B"


def generate_report(manifest_path, output_dir, formats=None):
    if formats is None:
        formats = ["json", "html"]
    with open(manifest_path, "r") as f:
        manifest = json.load(f)
    results = {}
    if "json" in formats:
        json_out = os.path.join(output_dir, "migration_report.json")
        shutil.copy2(manifest_path, json_out)
        results["json"] = json_out
    if "html" in formats:
        html_out = os.path.join(output_dir, "migration_report.html")
        results["html"] = generate_html_report(manifest, html_out)["path"]
    return {"success": True, "outputs": results}


if __name__ == "__main__":
    import sys
    import shutil
    manifest_path = sys.argv[1] if len(sys.argv) > 1 else "Manifest.json"
    output_dir = sys.argv[2] if len(sys.argv) > 2 else "."
    result = generate_report(manifest_path, output_dir)
    print(json.dumps(result, indent=2))
