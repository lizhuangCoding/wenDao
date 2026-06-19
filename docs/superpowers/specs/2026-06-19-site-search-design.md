# Site Search Design

## Goal

Build a first version of site search that is more useful than the current homepage list filter. Users should be able to search published articles from a dedicated page, narrow results by category and tag, and open matching articles quickly.

## Scope

V1 includes:

- A public `/search` route in the frontend.
- A backend search endpoint under `/api/search/articles`.
- Keyword matching across article title, summary, content, category name, and tag name.
- Optional category and tag filters.
- Pagination.
- Lightweight result snippets with highlighted keyword matches.
- Homepage search box submits to `/search?q=...` instead of only filtering the homepage list.

V1 does not include:

- External search engines such as Meilisearch or Elasticsearch.
- AI or vector semantic search.
- User search history or analytics.
- Admin-only draft search.

## Backend Design

Add a focused article search method near the existing article repository/service boundaries instead of overloading the existing article list endpoint further.

The repository will expose a `SearchArticles` query using SQL joins:

- `articles` is filtered to `status = published`.
- `categories` is joined for category-name matching.
- `article_tags` and `tags` are joined for tag-name matching.
- Keyword matching uses `LIKE` against title, summary, content, category name, and tag name.
- Results are grouped by article ID to avoid duplicates when multiple tags match.
- Ordering prioritizes title matches, then summary matches, then recent publish/create time.

The handler returns a paginated response shaped like:

```json
{
  "data": [
    {
      "article": {},
      "snippet": "...",
      "matched_fields": ["title", "content"]
    }
  ],
  "total": 12,
  "page": 1,
  "pageSize": 10,
  "totalPages": 2
}
```

Snippets are generated server-side from summary/content using a small helper that strips excessive whitespace and highlights the first keyword occurrence with `<mark>`. The frontend will render snippets as sanitized plain HTML from this trusted server helper only.

## Frontend Design

Add `frontend/src/pages/Search.tsx` and route `/search`.

The page contains:

- A search input initialized from `q`.
- Category selector.
- Tag selector.
- Result count and empty/loading/error states.
- Result cards with category, tags, title, snippet, and date.
- Pagination synced with query params.

Homepage search submits to `/search?q=<keyword>`. Category/tag buttons on the homepage continue to filter the homepage article list; the search page owns richer filtering.

## Error Handling

- Empty query still shows the search page, but no backend request is made until a keyword or filter exists.
- Backend validates page/pageSize through existing pagination helpers.
- Invalid filter IDs are treated as no match, returning an empty result set rather than a hard error.
- Database errors return the existing internal-error response style.

## Testing

Backend:

- Add focused tests for snippet generation and search filter construction where practical.
- Run `go test ./...`.

Frontend:

- Add source-level tests for the new route and homepage search navigation.
- Run `npm run lint`, `npm run build`, and `npm run test`.

## Implementation Notes

This design deliberately keeps search in MySQL for V1. The project is still a personal/article site, and a SQL implementation is enough for the current content size while keeping deployment simple. If search volume or ranking quality becomes a bottleneck, this endpoint can later be backed by a dedicated search engine without changing the frontend contract.
