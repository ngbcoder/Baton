#!/usr/bin/env python3
"""Build helper for the Baton desktop client."""

import argparse
import os
import platform
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
BIN_DIR = ROOT / "bin"


def run(cmd, env=None):
    print("$", " ".join(cmd))
    result = subprocess.run(cmd, cwd=ROOT, env=env)
    if result.returncode != 0:
        raise SystemExit(result.returncode)


def main():
    parser = argparse.ArgumentParser(description="Build Baton desktop")
    parser.add_argument("--target", choices=["windows", "linux"], default=None,
                        help="Target OS (default: current OS)")
    parser.add_argument("--test", action="store_true", help="Run go test before building")
    parser.add_argument("--clean", action="store_true", help="Remove bin/ first")
    args = parser.parse_args()

    target = args.target or ("windows" if platform.system() == "Windows" else "linux")
    env = os.environ.copy()
    env["CGO_ENABLED"] = "0"
    env["GOOS"] = target
    env["GOARCH"] = "amd64"

    if args.clean and BIN_DIR.exists():
        for p in BIN_DIR.iterdir():
            if p.is_file():
                p.unlink()

    BIN_DIR.mkdir(exist_ok=True)
    exe = "baton.exe" if target == "windows" else "baton"

    if args.test:
        run(["go", "test", "./..."], env=env)

    run(["go", "build", "-trimpath", "-ldflags", "-s -w", "-o", str(BIN_DIR / exe), "."], env=env)
    print(f"Built {BIN_DIR / exe}")


if __name__ == "__main__":
    try:
        main()
    except SystemExit as e:
        sys.exit(e.code)
    except Exception as e:
        print("build failed:", e, file=sys.stderr)
        sys.exit(1)
