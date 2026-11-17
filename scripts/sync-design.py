"""Update the two public CSS assets and their lock from a private design studio."""

import argparse
import hashlib
import json
from pathlib import Path


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("studio", type=Path, help="path to the design studio root")
    studio = parser.parse_args().studio

    digest = hashlib.sha256()
    for source in sorted((studio / "design").rglob("*")):
        if source.is_file() and not source.name.startswith("."):
            digest.update(source.relative_to(studio).as_posix().encode())
            digest.update(source.read_bytes())
    revision = digest.hexdigest()[:12]
    catalog = (studio / "studio/catalog.js").read_text(encoding="utf-8")
    prepared = json.loads(catalog.strip().removeprefix("window.STUDIO = ").removesuffix(";"))
    if prepared["revision"] != revision:
        parser.error("the studio catalog is stale, run scripts/prepare.py in the studio")

    assets = studio / "design/system"
    tokens = (assets / "tokens.css").read_bytes()
    first, _, bundle = (assets / "bundle.css").read_bytes().partition(b"\n")
    if not first.startswith(b"@import url('https://fonts.googleapis.com/"):
        parser.error("expected the Google Fonts import at the top of bundle.css")
    css = {"tokens.css": tokens, "bundle.css": bundle.removeprefix(b"\n")}

    web = Path(__file__).resolve().parents[1] / "web"
    for name, content in css.items():
        (web / "static" / name).write_bytes(content)
    lock = {
        "studio_revision": revision,
        "css_sha256": {
            name: hashlib.sha256(content).hexdigest() for name, content in css.items()
        },
    }
    (web / "design.lock.json").write_text(json.dumps(lock, indent=2) + "\n", encoding="utf-8")
    print(f"Synced public CSS from design revision {revision}")


if __name__ == "__main__":
    main()
