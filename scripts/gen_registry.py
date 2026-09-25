#!/usr/bin/env python3
"""按 dist/ 里的产物生成/更新 store/registry.json（供 CPA 自定义商店源使用）。"""
import hashlib, json, os, pathlib

TAG = os.environ.get("TAG", "").strip()
REPO = os.environ.get("REPO", "").strip()
VERSION = os.environ.get("VERSION", TAG.lstrip("v")).strip() or "0.0.0-dev"
ROOT = pathlib.Path(__file__).resolve().parent.parent
DIST, STORE = ROOT / "dist", ROOT / "store"
STORE.mkdir(exist_ok=True)
REGISTRY = STORE / "registry.json"

PLUGIN = {
    "id": "bps-usage",
    "name": "BPS 用量",
    "description": "在 CPAMC 中查看 bps-shim 采集的用量：BPS 直连与转发 CPA 两条路径的请求数、tokens（含缓存/推理）、TTFT、耗时与工具调用。",
    "author": "xulizheng66",
    "repository": f"https://github.com/{REPO}" if REPO else "",
    "homepage": f"https://github.com/{REPO}" if REPO else "",
    "license": "MIT",
    "tags": ["监控", "bps", "usage"],
}

def artifacts():
    out = []
    for f in sorted(DIST.glob("*.zip")):   # 商店安装需要 zip（内部 <id>-v<version>.so）
        name = f.name
        arch = "arm64" if "arm64" in name else ("amd64" if "amd64" in name else None)
        if not arch:
            continue
        out.append({
            "goos": "linux", "goarch": arch,
            "url": f"https://github.com/{REPO}/releases/download/{TAG}/{name}" if REPO and TAG else name,
            "size": f.stat().st_size,
            "sha256": hashlib.sha256(f.read_bytes()).hexdigest(),
        })
    return out

def main():
    arts = artifacts()
    if not arts:
        raise SystemExit("dist/ 下没有 .zip 产物")
    entry = dict(PLUGIN)
    entry["version"] = VERSION
    entry["install"] = {"type": "direct", "artifacts": arts}
    entry["versions"] = [{"version": VERSION, "install": entry["install"]}]

    reg = {"schema_version": 2, "plugins": []}
    if REGISTRY.exists():
        try:
            reg = json.loads(REGISTRY.read_text())
        except Exception:
            pass
    plugins = [p for p in reg.get("plugins", []) if p.get("id") != entry["id"]]
    prev = next((p for p in reg.get("plugins", []) if p.get("id") == entry["id"]), None)
    if prev:
        history = [v for v in prev.get("versions", []) if v.get("version") != VERSION]
        entry["versions"] = [entry["versions"][0]] + history
    reg["schema_version"] = 2
    reg["plugins"] = [entry] + plugins
    REGISTRY.write_text(json.dumps(reg, ensure_ascii=False, indent=2) + "\n")
    print(f"registry 已更新: {REGISTRY} ({len(arts)} artifacts, version {VERSION})")

if __name__ == "__main__":
    main()
