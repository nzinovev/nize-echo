# Database schema

How the concept's content model (*Personal Site — Concept*, "Content model") maps to Postgres tables. The DDL is in `migrations/`; schema v1 is `00002_schema_v1.sql`.

## Entities

| Concept entity | Tables | Notes |
| --- | --- | --- |
| Post | `posts` | Status, publish time, cover, created and updated times. Language-independent. |
| Post translation | `post_translations` | Key `(post_id, lang)`, so a post has at most one translation per language. Slug is unique per language. Stores `body_md`, `body_html` (rendered on save) and `reading_minutes`. |
| Post's context | `context_posts` | Key `post_id`, so a post is in at most one context; `position` orders posts inside the context. |
| Post's tags | `post_tags` | Many-to-many. |
| Context | `contexts`, `context_translations` | Slug, cover, site order (`position`) and `visible` on `contexts`; title and description per language. |
| Tag | `tags`, `tag_labels` | Slug on `tags`; label per language. |
| Page | `pages`, `page_translations`, `page_files` | `pages.key` names the page (`cv`). Markdown, HTML and SEO fields per language; `fields` (jsonb) holds page-specific extras such as the CV's email and location. `page_files` attaches one file (the CV PDF) per language. |
| Media | `media`, `media_variants`, `media_alt_texts` | Original file with storage key, MIME type, size and dimensions (none for PDFs); WebP variants keyed by width; alt text per language. `private` leaves room for signed URLs. |
| Slug history | `post_slug_history`, `context_slug_history` | Old slugs that 301 to the current one. Post slugs are per language. |
| Admin user | `users` | Usernames are unique ignoring case. Not admin-specific, so reader accounts can share it later. |
| Session | `sessions` | Server-side sessions; only the token's hash is stored. Revoking deletes the row. |
| Settings | `settings` | Key → jsonb value. |

## Post status

`post_status` is `draft`, `published` or `unpublished`. Scheduled is not stored:

| Concept state | Row |
| --- | --- |
| Draft | `status = 'draft'` |
| Scheduled | `status = 'published' AND publish_at > now()` |
| Published | `status = 'published' AND publish_at <= now()` |
| Unpublished | `status = 'unpublished'`, `unpublished_at` set |

Public queries filter `status = 'published' AND publish_at <= now()`, so a scheduled post goes live with no background job. A published post must have `publish_at`. An unpublished post keeps its `publish_at`, so `unpublished_at - publish_at` tells whether it was public for more than a day (410 rather than 404).

## Conventions

- IDs are `bigint GENERATED ALWAYS AS IDENTITY`; times are `timestamptz`. Queries set `updated_at`; there are no triggers.
- Languages use the `lang` enum (`ru`, `en`). Per-language fields live in a child table keyed `(parent_id, lang)`.
- Slugs are lowercase ASCII words joined by single hyphens, enforced by a `CHECK`.
- Media referenced by a post cover, a context cover or a page file can't be deleted: the foreign keys have no `ON DELETE` action.
- Order columns (`contexts.position`, `context_posts.position`) are unique and deferred, so a reorder can swap positions inside one transaction.
- Deleting a parent row cascades to its translations, links and slug history; deleting a post or tag never deletes the other side of `post_tags`.
