import urllib.request
import json
import os

with open("token.txt") as f:
    token = f.read().strip()

headers = {"Authorization": f"token {token}", "User-Agent": "rokboll"}

req = urllib.request.Request("https://api.github.com/repos/rokbollyt/rokbollvpn/releases/tags/latest", headers=headers)
with urllib.request.urlopen(req) as resp:
    rel_data = json.loads(resp.read().decode())

upload_url = rel_data["upload_url"].split("{")[0]
assets = rel_data.get("assets", [])

# Delete old IPA if present
for a in assets:
    if a["name"] in ("HappPort-TrollStore.ipa", "Happ.ipa", "RokbollVPN.ipa"):
        asset_id = a["id"]
        print(f"Deleting old asset {a['name']} (ID {asset_id})...")
        del_req = urllib.request.Request(
            f"https://api.github.com/repos/rokbollyt/rokbollvpn/releases/assets/{asset_id}",
            headers=headers,
            method="DELETE"
        )
        with urllib.request.urlopen(del_req) as del_resp:
            print("Deleted.")

ipa_path = "HappPort-TrollStore.ipa"
ipa_size = os.path.getsize(ipa_path)
print(f"Uploading {ipa_path} ({ipa_size} bytes)...")

with open(ipa_path, "rb") as f:
    ipa_bytes = f.read()

upload_headers = {
    "Authorization": f"token {token}",
    "User-Agent": "rokboll",
    "Content-Type": "application/octet-stream",
    "Content-Length": str(len(ipa_bytes))
}

upload_req = urllib.request.Request(
    f"{upload_url}?name=HappPort-TrollStore.ipa",
    data=ipa_bytes,
    headers=upload_headers,
    method="POST"
)

with urllib.request.urlopen(upload_req) as up_resp:
    res_data = json.loads(up_resp.read().decode())
    print("Upload complete!")
    print("Direct Download URL:", res_data.get("browser_download_url"))

