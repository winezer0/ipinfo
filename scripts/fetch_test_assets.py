"""Fetch pinned integration databases. Python 3 standard library only."""
import argparse
import concurrent.futures
import hashlib
import json
import os
from pathlib import Path
import tempfile
import time
import urllib.request


ROOT = Path(__file__).resolve().parent.parent


def checksum(path):
    sha256 = hashlib.sha256()
    with path.open("rb") as stream:
        for chunk in iter(lambda: stream.read(1024 * 1024), b""):
            sha256.update(chunk)
    return sha256.hexdigest()


def matches(path, entry):
    return checksum(path) == entry["sha256"]


def fetch(entry, dest):
    name = entry["name"]
    if Path(name).name != name or name in (".", ".."):
        raise ValueError(f"invalid filename: {name}")
    expected = entry.get("sha256", "")
    if len(expected) != 64 or any(c not in "0123456789abcdef" for c in expected):
        raise ValueError(f"missing or invalid SHA-256: {name}")
    target = dest / name
    if target.is_file() and matches(target, entry):
        return f"verified {name} sha256={entry['sha256']}"
    for attempt in range(3):
        temp = None
        try:
            request = urllib.request.Request(entry["url"], headers={"User-Agent": "ipinfo-integration-tests"})
            with urllib.request.urlopen(request, timeout=30) as response:
                with tempfile.NamedTemporaryFile(dir=dest, prefix=name + ".", suffix=".part", delete=False) as output:
                    temp = Path(output.name)
                    for chunk in iter(lambda: response.read(1024 * 1024), b""):
                        output.write(chunk)
            if not matches(temp, entry):
                raise ValueError(f"checksum mismatch: {name}; upstream may have replaced the pinned release")
            os.replace(temp, target)
            return f"downloaded {name} sha256={entry['sha256']}"
        except Exception:
            if attempt == 2:
                raise
            time.sleep(attempt + 1)
        finally:
            if temp is not None:
                temp.unlink(missing_ok=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dir", type=Path, default=ROOT / "assets")
    parser.add_argument("--only", nargs="+", help="fetch selected filenames")
    args = parser.parse_args()
    entries = json.loads((ROOT / "scripts/test-assets.json").read_text(encoding="utf-8"))
    if args.only:
        unknown = set(args.only) - {e["name"] for e in entries}
        if unknown:
            parser.error(f"unknown assets: {sorted(unknown)}")
        entries = [e for e in entries if e["name"] in args.only]
    args.dir.mkdir(parents=True, exist_ok=True)
    failures = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=3) as pool:
        jobs = {pool.submit(fetch, e, args.dir): e["name"] for e in entries}
        for job in concurrent.futures.as_completed(jobs):
            try:
                print(job.result(), flush=True)
            except Exception as error:
                failures.append(jobs[job])
                print(f"FAILED {jobs[job]}: {error}", flush=True)
    if failures:
        raise SystemExit(f"failed assets: {', '.join(failures)}")


if __name__ == "__main__":
    main()
