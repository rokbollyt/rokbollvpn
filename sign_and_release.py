import os
import subprocess
import urllib.request
import json

zipalign_exe = r"C:\Users\RokBolL\AppData\Local\Android\Sdk\build-tools\34.0.0\zipalign.exe"
apksigner_bat = r"C:\Users\RokBolL\AppData\Local\Android\Sdk\build-tools\34.0.0\apksigner.bat"
keystore = r"D:\projects\go\debug.keystore"

unaligned = r"D:\projects\go\Happ-unaligned.apk"
aligned = r"D:\projects\go\Happ-aligned.apk"
signed = r"D:\projects\go\Happ-fixed.apk"

if not os.path.exists(unaligned):
    print("Happ-unaligned.apk does not exist!")
    exit(1)

print("Running zipalign...")
if os.path.exists(aligned):
    os.remove(aligned)
res = subprocess.run([zipalign_exe, "-p", "-f", "-v", "4", unaligned, aligned], capture_output=True, text=True)
if res.returncode != 0:
    print("zipalign failed:", res.stderr)
    exit(1)
print("zipalign succeeded.")

print("Running apksigner...")
if os.path.exists(signed):
    os.remove(signed)
res = subprocess.run([
    apksigner_bat, "sign",
    "--ks", keystore,
    "--ks-pass", "pass:android",
    "--key-pass", "pass:android",
    "--out", signed,
    aligned
], capture_output=True, text=True)
if res.returncode != 0:
    print("apksigner failed:", res.stderr)
    exit(1)
print("apksigner succeeded.")

# Verify signature
res = subprocess.run([apksigner_bat, "verify", signed], capture_output=True, text=True)
print("Signature verification:", res.stdout if res.returncode == 0 else res.stderr)

# GitHub Release upload
token = os.environ.get("GITHUB_TOKEN", "")
if not token and os.path.exists("token.txt"):
    with open("token.txt", "r") as f:
        token = f.read().strip()
headers = {
    "Authorization": f"token {token}",
    "User-Agent": "rokboll"
}

# 1. Get release info and find existing Happ-MAX-Release.apk asset
print("Fetching release info...")
req = urllib.request.Request("https://api.github.com/repos/rokbollyt/rokbollvpn/releases/tags/latest", headers=headers)
with urllib.request.urlopen(req) as resp:
    rel_data = json.loads(resp.read().decode())

upload_url = rel_data["upload_url"].split("{")[0]
assets = rel_data.get("assets", [])

for a in assets:
    if a["name"] == "Happ-MAX-Release.apk":
        asset_id = a["id"]
        print(f"Deleting old asset {asset_id}...")
        del_req = urllib.request.Request(
            f"https://api.github.com/repos/rokbollyt/rokbollvpn/releases/assets/{asset_id}",
            headers=headers,
            method="DELETE"
        )
        with urllib.request.urlopen(del_req) as del_resp:
            print("Deleted old asset.")

# 2. Upload new asset
print(f"Uploading new Happ-MAX-Release.apk ({os.path.getsize(signed)} bytes)...")
with open(signed, "rb") as f:
    apk_bytes = f.read()

upload_headers = {
    "Authorization": f"token {token}",
    "User-Agent": "rokboll",
    "Content-Type": "application/vnd.android.package-archive",
    "Content-Length": str(len(apk_bytes))
}

upload_req = urllib.request.Request(
    f"{upload_url}?name=Happ-MAX-Release.apk",
    data=apk_bytes,
    headers=upload_headers,
    method="POST"
)

with urllib.request.urlopen(upload_req) as up_resp:
    res_data = json.loads(up_resp.read().decode())
    print("Upload complete!")
    print("Download URL:", res_data.get("browser_download_url"))
