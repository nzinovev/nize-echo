# Design references

The design studio is maintained separately from this public repository. Paths below are relative to the studio root.

| Reference | Studio path |
| --- | --- |
| Product behavior, architecture and acceptance criteria | `docs/concept.md` |
| Task scope, dependencies and definition of done | `docs/backlog.md` |
| Night theme, tokens and component rules | `design/system/README.md` and `design/system/components/` |
| Public and admin layouts | `design/mockups/` |

Use the concept for behavior and the design system for appearance. Include the studio revision and the relevant private references in UI handoffs.

## Update the shipped CSS

Refresh the catalog by running `python3 scripts/prepare.py` in the studio. Run the following commands from the product repository.

```sh
python3 scripts/sync-design.py /path/to/studio
make assets
make test
```

The sync copies only `tokens.css` and `bundle.css` into `web/static/`. It strips the studio's Google Fonts import from `bundle.css` because the site uses local fonts. Commit the copied CSS, `web/design.lock.json` and `web/static-assets.json`. CI compares CSS against the locked SHA‑256 digests and checks that the embedded manifest matches the public assets. The tests run from a clean checkout with the lock as their reference.

Open `/styleguide` to compare the rendered CSS with the private Post mockups at desktop and phone widths. Keep the studio source, its exports and mockups outside this repository.
