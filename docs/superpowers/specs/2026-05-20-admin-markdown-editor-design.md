# Admin Markdown Editor Enhancement Design

## Overview
Improve the admin article editing experience by turning the current plain Markdown textarea and right-side preview into a more polished writing workspace.

The first version stays intentionally lightweight:
- keep the existing `textarea` as the editing engine,
- keep the existing `ReactMarkdown` preview renderer,
- add editor controls, layout modes, richer status information, and dedicated admin preview styling.

This avoids a large editor dependency while making the existing workflow feel more deliberate and useful.

## Scope
### In scope
- Admin article editor Markdown content area in `frontend/src/views/admin/articles/ArticleEditor.tsx`.
- The right-side preview shown while editing an article.
- Markdown insertion helpers for toolbar actions.
- Frontend tests for Markdown text insertion behavior.

### Out of scope
- Public article detail rendering.
- Backend article APIs.
- Markdown storage format.
- A rich text or WYSIWYG editor.
- New upload endpoints.

## Current Problems
### Plain input surface
The current Markdown content field is a generic textarea with fixed height. It works, but it does not feel like a focused authoring tool.

### Weak editing affordances
Authors need to manually type common Markdown syntax for headings, bold text, links, lists, quotes, dividers, and code blocks.

### Preview feels like a passive box
The preview currently sits in a simple scroll container. It lacks a clear header, empty state, and admin-specific visual treatment.

### Layout is fixed
The editor always shows a two-column split on large screens. Authors cannot switch to a focused writing mode or full preview mode.

## Design Summary
Replace the content section with an admin Markdown studio embedded inside the existing article form.

The studio will provide:
- toolbar buttons for common Markdown insertions,
- three view modes: edit, split, and preview,
- improved editor chrome with a title bar and document statistics,
- a dedicated preview panel with empty state and more polished Markdown styling,
- continued support for existing image upload, pasted image upload, auto-save, and lazy-loaded preview.

## Architecture
### Existing form ownership stays in `ArticleEditor`
`ArticleEditor` will continue owning:
- article form state,
- auto-save state,
- image upload calls,
- summary generation,
- validation and submit behavior.

The Markdown studio will receive controlled props:
- `content`,
- `onContentChange`,
- `textareaRef`,
- `onPaste`,
- `onImageUploadClick`.

This keeps persistence and API behavior unchanged.

### Markdown insertion helpers
Add a small pure utility module for editor text operations.

Recommended file:
- `frontend/src/utils/markdownEditor.ts`

Core API:
- `applyMarkdownAction(input, action)` returns the next text and cursor selection.

The helper should support these actions:
- heading,
- bold,
- quote,
- unordered list,
- ordered list,
- code block,
- inline code,
- link,
- divider.

Image insertion remains owned by the existing upload flow instead of the pure text helper.

The UI layer will read the textarea selection, call the helper, update controlled content, then restore focus and selection.

### Admin-only preview styling
Add admin preview styling either through Tailwind classes in the component or a small CSS section in `frontend/src/styles/index.css`.

The styling must be scoped to the admin editor preview so public article rendering is not affected.

## UI Design
### Toolbar
The toolbar sits above the editor and preview panels.

Controls:
- mode segmented control: edit, split, preview,
- Markdown action buttons with icons from `lucide-react`,
- image insertion button reusing the existing file input flow,
- compact stats showing characters, words or approximate reading time.

The toolbar should be dense and utilitarian, not a marketing-style card.

### Editor panel
The editor panel contains:
- a small header with "Markdown" and the current save state,
- the textarea,
- footer metadata such as line count and character count.

Visual direction:
- neutral background,
- monospace text,
- stable height around the current 500px baseline,
- clear focus ring,
- no nested card-heavy layout.

### Preview panel
The preview panel contains:
- a header with "Preview",
- a scrollable rendered Markdown body,
- an empty state when content is blank.

The preview should make code blocks, blockquotes, tables, links, and images easier to inspect while still looking like an editing preview rather than the public article page.

### View modes
- `split`: default on desktop, editor and preview side by side.
- `edit`: editor only, full width.
- `preview`: preview only, full width.

On narrow screens, the selected mode still applies, but split mode stacks panels vertically.

## Data Flow
1. User clicks a toolbar action.
2. Component reads `selectionStart` and `selectionEnd` from the textarea.
3. `applyMarkdownAction` computes the next content and selection.
4. `ArticleEditor` updates `formData.content`.
5. The textarea selection is restored after React updates the value.
6. Existing auto-save and local draft effects observe the changed form data as they do today.
7. Preview receives the same content and re-renders through `ArticlePreview`.

## Error Handling
### Toolbar actions
Toolbar actions should always be local and deterministic. They should not show error toasts.

If no text is selected, actions insert a sensible placeholder or syntax skeleton.

### Image upload
The existing image upload error path remains unchanged:
- upload failures show the current toast,
- editor content stays unchanged if upload fails.

### Preview rendering
Preview rendering keeps the existing lazy import and Suspense fallback.

If content is empty, show an empty state instead of a blank panel.

## Testing Plan
### Unit tests
Add tests for `frontend/src/utils/markdownEditor.ts`.

Coverage:
- wrapping selected text with bold syntax,
- inserting heading syntax at the current line,
- converting selected multi-line text to list items,
- inserting a fenced code block,
- preserving cursor placement for inserted placeholders.

Use the existing lightweight Node test pattern under `frontend/src/utils/*.test.mjs`.

### Build and lint
Run from `frontend/`:
- `npm run build`
- `npm run lint`

### Manual verification
Start Vite and verify:
- editor loads on the admin article edit page,
- toolbar actions modify content correctly,
- existing image file insertion still works,
- existing pasted image upload still works,
- edit, split, and preview modes switch without layout overlap,
- dark mode remains readable.

## Acceptance Criteria
- The admin article Markdown input feels like a dedicated writing surface rather than a plain textarea.
- Common Markdown syntax can be inserted from toolbar buttons.
- The right-side preview has a clear header, empty state, and improved admin-only rendering styles.
- Edit, split, and preview modes work on desktop and mobile widths.
- Existing auto-save, local draft, image upload, pasted image upload, and article save behavior remain unchanged.
- Public article rendering is not changed.
