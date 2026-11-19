"""Download a snapshot of the public design catalog into a local server cache."""
import argparse
from concurrent.futures import ThreadPoolExecutor
import hashlib
from io import BytesIO
import json
import os
from pathlib import Path
import re
import tempfile
import time
from urllib.error import HTTPError, URLError
from urllib.parse import urlencode, urljoin, urlsplit
from urllib.request import HTTPRedirectHandler, build_opener
from zipfile import BadZipFile, ZipFile

PREFIX = "/api/v1/design-resources/public-cache"
ASSET_PREFIX = PREFIX + "/assets/"
MODES = ("image", "prototype", "deck", "video", "web", "logo")
MEDIA = {"image/jpeg": "jpg", "image/png": "png", "image/webp": "webp",
         "image/avif": "avif", "image/gif": "gif", "video/mp4": "mp4",
         "video/webm": "webm", "application/zip": "zip",
         "application/x-zip-compressed": "zip", "application/octet-stream": "zip"}

class NoRedirect(HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None

def fetch(url, limit):
    parsed = urlsplit(url)
    if parsed.scheme != "https" or parsed.netloc not in ALLOWED or parsed.query and parsed.hostname != urlsplit(ORIGIN).hostname:
        raise ValueError("URL outside the public catalog")
    for attempt in range(3):
        try:
            with build_opener(NoRedirect()).open(url, timeout=30) as response:
                body = response.read(limit + 1)
                if len(body) > limit:
                    raise ValueError("response exceeds cache size limit")
                return body, response.headers.get_content_type()
        except (OSError, URLError) as exc:
            if isinstance(exc, HTTPError) and exc.code < 500 and exc.code != 429:
                raise
            if attempt == 2:
                raise
            time.sleep(attempt + 1)

def catalog(path, **query):
    body, _ = fetch(BASE + "/catalog/" + path + "?" + urlencode(query), 8 << 20)
    response = json.loads(body)
    if response.get("code") != 0 or not isinstance(response.get("data"), dict):
        raise ValueError("invalid public catalog response")
    return response["data"]

def pages(kind, page_size, **filters):
    page = 1
    while True:
        data = catalog(kind, page=page, page_size=page_size, **filters)
        for item in data["items"]:
            yield item, data["asset_snapshot_id"]
        if page >= data["total_pages"]:
            return
        page += 1

def write_atomic(destination, body):
    with tempfile.NamedTemporaryFile(dir=destination.parent, delete=False) as temp:
        temp.write(body)
    os.replace(temp.name, destination)

LEGACY_URL = re.compile(rb"https?://[a-z0-9.-]*(?:baizhi\.cloud|monkeycode-ai\.com|chaitin\.(?:com|cn))(?=[:/\s\"']|$)", re.I)

def should_cache_archive(body):
    try:
        with ZipFile(BytesIO(body)) as archive:
            inspected = 0
            for member in archive.infolist():
                if Path(member.filename).suffix.lower() not in (".html", ".htm", ".js", ".css", ".json", ".md", ".svg"):
                    continue
                inspected += member.file_size
                if member.file_size > 2 << 20 or inspected > 20 << 20 or LEGACY_URL.search(archive.read(member)):
                    return False
    except (BadZipFile, OSError, RuntimeError, ValueError):
        return False
    return True

def verify_cache(root, bundle):
    if not bundle["revision"]:
        raise ValueError("cache catalog snapshot ID is missing")
    failures = []
    archives = 0
    files = 0
    for kind, rows in bundle["details"].items():
        for key, detail in rows.items():
            listing = bundle["listing"][kind][key]
            urls = {url for row in (detail, listing)
                    for url in (row.get("download_url"), (row.get("download") or {}).get("url"))
                    if isinstance(url, str) and url.startswith(ASSET_PREFIX)}
            declared = detail.get("files") or []
            for url in sorted(urls):
                label = f"{kind}/{key}"
                name = url[len(ASSET_PREFIX):]
                if not re.fullmatch(r"[a-f0-9]{64}\.zip", name):
                    failures.append(f"{label}: invalid ZIP asset name")
                    continue
                if kind == "templates" and not declared:
                    failures.append(f"{label}: downloadable template has no file manifest")
                    continue
                archives += 1
                try:
                    body = (root / "assets" / name).read_bytes()
                    if hashlib.sha256(body).hexdigest() != name[:-4]:
                        failures.append(f"{label}: ZIP SHA-256 mismatch")
                        continue
                    prefix = ("catalog-template-" if kind == "templates" else "design-system-") + key + "/"
                    with ZipFile(BytesIO(body)) as archive:
                        members = [entry for entry in archive.infolist() if not entry.is_dir()]
                        wrapped = all(entry.filename.startswith(prefix) for entry in members)
                        entries = {entry.filename[len(prefix):] if wrapped else entry.filename: entry
                                   for entry in members}
                        if not members or len(entries) != len(members):
                            failures.append(f"{label}: empty ZIP or duplicate file paths")
                            continue
                        if not declared and (damaged := archive.testzip()) is not None:
                            failures.append(f"{label}: damaged ZIP member {damaged}")
                            continue
                        for file in declared:
                            path = file["path"]
                            entry = entries.get(path)
                            if entry is None and kind == "design-systems" and wrapped and path.startswith(prefix):
                                entry = entries.get(path[len(prefix):])
                            if entry is None:
                                failures.append(f"{label}: missing file {path}")
                                continue
                            expected = file["sha256"].removeprefix("sha256:").lower()
                            if entry.file_size != file["size"] or hashlib.sha256(archive.read(entry)).hexdigest() != expected:
                                failures.append(f"{label}: file size or SHA-256 mismatch: {path}")
                            files += 1
                except (BadZipFile, OSError, RuntimeError, ValueError) as exc:
                    failures.append(f"{label}: cannot verify ZIP: {exc}")
    if failures:
        raise ValueError("\n".join(failures))
    print(f"Verified {archives} ZIP archives and {files} declared files", flush=True)

def synchronize(root):
    assets = root / "assets"
    assets.mkdir(parents=True, exist_ok=True, mode=0o700)
    groups = {"templates": {}, "design-systems": {}}
    mode_buckets = {}
    revisions = set()
    for item, revision in pages("design-systems", 100):
        groups["design-systems"][item["id"]] = item
        revisions.add(revision)
    for mode in ("", *MODES):
        bucket = []
        try:
            for item, revision in pages("templates", 20, **({"mode": mode} if mode else {})):
                groups["templates"][item["id"]] = item
                revisions.add(revision)
                bucket.append(item["id"])
        except HTTPError as exc:
            if not mode or exc.code != 400:
                raise
            continue
        mode_buckets[mode] = bucket
    for category in ("posters-typography", "logo", "products-ecommerce"):
        bucket = []
        for item, revision in pages("templates", 20, mode="image", category=category):
            groups["templates"][item["id"]] = item
            revisions.add(revision)
            bucket.append(item["id"])
        mode_buckets["image/" + category] = bucket
    if len(revisions) != 1:
        raise RuntimeError("public catalog changed during download")
    revision = revisions.pop()
    print(f"Public index: {len(groups['templates'])} cases, {len(groups['design-systems'])} design systems", flush=True)

    def cache_asset(value, archive):
        if not value:
            return value
        url = urljoin(ORIGIN, value)
        try:
            body, mime = fetch(url, (36 if archive else 16) << 20)
            suffix = MEDIA.get(mime)
            if archive and suffix == "zip" and should_cache_archive(body) or not archive and suffix and suffix != "zip":
                name = hashlib.sha256(body).hexdigest() + "." + suffix
                if not (assets / name).exists():
                    write_atomic(assets / name, body)
                return PREFIX + "/assets/" + name
        except (OSError, ValueError):
            pass
        # Keep the original URL for a public asset that could not be cached.
        return url

    def prepare(entry):
        kind, item = entry
        key = item["id"]
        if not re.fullmatch(r"[A-Za-z0-9_.-]{1,160}", key):
            raise ValueError("invalid resource ID")
        try:
            full = catalog(kind + "/" + key)
        except (OSError, ValueError):
            full = dict(item)
        if full.get("asset_snapshot_id", revision) != revision or full.get("content_digest") != item.get("content_digest"):
            raise RuntimeError("public resource changed during download")
        fields = {}
        for name in ("preview_url", "download_url"):
            if full.get(name) or item.get(name):
                fields[name] = cache_asset(full.get(name) or item[name], name == "download_url")
        for name, paths in {"card_preview": ("poster_url", "video_url"), "preview": ("url",), "download": ("url",)}.items():
            mapping = full.get(name) or item.get(name)
            if isinstance(mapping, dict):
                fields[name] = {**mapping, **{path: cache_asset(mapping[path], name == "download") for path in paths if mapping.get(path)}}
        metadata = full.get("metadata") or {}
        display = {name: full.get(name) or metadata[name] for name in ("category", "colors", "capabilities") if full.get(name) or name in metadata}
        return kind, key, {**item, **fields, **display}, {**full, **fields, **display}

    listing = {kind: {} for kind in groups}
    details = {kind: {} for kind in groups}
    with ThreadPoolExecutor(max_workers=8) as workers:
        jobs = ((kind, item) for kind, items in groups.items() for item in items.values())
        for kind, key, item, detail in workers.map(prepare, jobs):
            listing[kind][key], details[kind][key] = item, detail
    bundle = {"revision": revision, "listing": listing, "details": details,
              "mode_buckets": mode_buckets}
    verify_cache(root, bundle)
    write_atomic(root / "catalog.json", json.dumps(bundle, ensure_ascii=False).encode())
    print(f"Published {sum(map(len, listing.values()))} resources and {len(list(assets.iterdir()))} local files", flush=True)

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("directory", type=Path)
    action = parser.add_mutually_exclusive_group(required=True)
    action.add_argument("--source", help="Administrator-approved HTTPS catalog API base")
    action.add_argument("--verify", action="store_true", help="Verify cached ZIP archives and their file manifests")
    parser.add_argument("--media-origin", action="append", default=[], help="Additional HTTPS asset origin")
    args = parser.parse_args()
    if args.verify:
        try:
            verify_cache(args.directory, json.loads((args.directory / "catalog.json").read_bytes()))
        except (KeyError, OSError, ValueError) as exc:
            parser.exit(1, f"{exc}\n")
        raise SystemExit(0)
    source = urlsplit(args.source)
    if source.scheme != "https" or not source.hostname or source.username or source.password or source.query or source.fragment:
        parser.error("source must be a credential-free HTTPS catalog URL")
    BASE = args.source.rstrip("/")
    ORIGIN = f"https://{source.netloc}"
    ALLOWED = {source.netloc}
    for value in args.media_origin:
        media = urlsplit(value)
        if media.scheme != "https" or not media.hostname or media.username or media.password or media.path not in ("", "/") or media.query or media.fragment:
            parser.error("media-origin must be a credential-free HTTPS origin")
        ALLOWED.add(media.netloc)
    synchronize(args.directory)
