# Markdown Writing Studio Design

## Overview
Improve the admin article Markdown editor with two focused upgrades:

- add a reusable font color control with preset colors and custom color selection,
- add an immersive writing mode that gives the content editor substantially more space.

The implementation should stay component-first. `ArticleEditor` will keep owning article data, persistence, upload behavior, validation, and navigation. A new editor component will own the writing surface, toolbar, preview panels, color insertion controls, and immersive layout state.

## Goals
- Let authors apply a chosen text color from common presets or a custom color picker.
- Make long-form article writing feel less cramped than the current narrow form layout.
- Preserve the current Markdown storage format and editing workflow.
- Keep existing auto-save, local draft, image upload, pasted image upload, preview, and save behavior unchanged.
- Reduce `ArticleEditor` complexity by moving Markdown studio UI into dedicated components.

## Non-Goals
- Do not replace the editor with a new Markdown or WYSIWYG engine.
- Do not change backend article APIs or database schema.
- Do not change public article rendering except by allowing already-supported inline HTML color spans to render.
- Do not add collaboration, block editing, slash commands, or document templates.

## Current State
The admin article editor currently uses:

- `frontend/src/views/admin/articles/ArticleEditor.tsx` for the full article form and Markdown editor UI,
- a controlled `textarea` as the editing engine,
- `frontend/src/utils/markdownEditor.ts` for pure Markdown insertion helpers,
- `ArticlePreview` with `react-markdown`, `remark-gfm`, `rehype-highlight`, and `rehype-raw`,
- `tdesign-react` for existing form controls such as `Select`.

This setup is lightweight and compatible with the requested feature. The main problem is that editor UI responsibilities are concentrated in `ArticleEditor`, and the writing area feels constrained inside the general form card.

## Recommended Approach
Use a new component boundary around the Markdown writing experience.

Recommended component:

- `frontend/src/views/admin/articles/components/MarkdownWritingStudio.tsx`

Optional smaller local components inside that file or nearby files:

- `MarkdownToolbar`
- `MarkdownColorControl`
- `MarkdownEditorPanel`
- `MarkdownPreviewPanel`

The first implementation can keep these helpers in one component file if that is clearer, but the boundaries should remain explicit.

## Component Responsibilities
### ArticleEditor
`ArticleEditor` remains responsible for:

- loading and mutating article data,
- title, category, status, summary, and cover image form fields,
- auto-save and local draft behavior,
- image upload API calls,
- submit validation and navigation,
- passing save state and stats into the writing studio.

`ArticleEditor` should no longer render the Markdown toolbar, editor panel, or preview panel directly.

### MarkdownWritingStudio
`MarkdownWritingStudio` owns:

- edit, split, and preview view modes,
- immersive writing mode toggle,
- toolbar rendering,
- color picking UI,
- reading textarea selection and applying Markdown text edits,
- preview and empty state rendering,
- editor and preview panel layout.

The component receives controlled data and callbacks:

- `content: string`
- `onContentChange(nextContent: string): void`
- `textareaRef: RefObject<HTMLTextAreaElement>`
- `onPaste(e): void`
- `onImageUploadClick(): void`
- stats such as character count, line count, word count, reading minutes
- save status such as last saved time and whether auto-save is running

## Font Color Behavior
Use inline HTML spans because the current renderer already supports raw HTML through `rehypeRaw`.

Inserted format:

```md
<span style="color: #ef4444">选中文字</span>
```

Behavior:

- If text is selected, wrap the selected text in a color span.
- If no text is selected, insert a color span with `彩色文字` and select the placeholder text.
- The color value should be normalized to a safe CSS color string before insertion.
- Prefer hex output for preset and custom colors.
- Do not show a toast for color insertion. It is a local deterministic action.

### Color UI
Use `tdesign-react` `ColorPicker`, not a custom color picker.

The control should provide:

- visible preset swatches in the toolbar for common colors,
- a custom picker for arbitrary colors,
- a compact current-color preview,
- accessible labels and hover tooltips consistent with existing toolbar buttons.

Suggested preset palette:

- red `#ef4444`
- orange `#f97316`
- amber `#f59e0b`
- green `#10b981`
- sky `#0ea5e9`
- indigo `#6366f1`
- pink `#ec4899`
- neutral `#525252`

## Immersive Writing Mode
Add a toolbar action named `专注写作`.

When enabled:

- The article page container expands from the current narrow editor width toward a display-width workspace.
- The Markdown editor receives the primary visual weight.
- The editor panel height becomes viewport-relative, around `calc(100vh - 220px)` with a sensible minimum.
- Metadata fields remain available but become compact, so the user can still save, publish, adjust category/status, and inspect cover/summary without leaving the page.
- The existing edit, split, and preview modes continue to work.

When disabled:

- The page returns to the normal article form layout.
- Existing field order and behavior should remain familiar.

Mobile behavior:

- Split mode stacks editor and preview vertically.
- Immersive mode still increases vertical room, but must not create horizontal scrolling.
- Toolbar controls wrap cleanly without text overlap.

## Layout Direction
Normal mode:

- Keep a form-first layout close to the current page.
- Increase the editor baseline height from about `540px` to about `640px`.
- Keep title/category/status/summary/cover above the writing studio.

Immersive mode:

- Use a wider outer container, ideally `max-w-display` or equivalent.
- Move metadata into a compact top band or side column.
- Give the writing studio most of the page width.
- In edit-only mode, the textarea should feel like a full writing desk, not a narrow column.
- In split mode, editor and preview should be balanced without either becoming unusably thin.

Avoid card-heavy nesting. The editor panels may be framed tools, but page sections should not become stacked decorative cards.

## Data Flow
1. User clicks a Markdown toolbar action.
2. `MarkdownWritingStudio` reads the textarea selection.
3. The pure helper in `markdownEditor.ts` computes the next text, replacement range, and target selection.
4. The studio updates content through `onContentChange`.
5. The textarea focus and selection are restored.
6. Existing auto-save and local draft effects in `ArticleEditor` observe the changed content.
7. Preview rerenders using the same `ArticlePreview` component.

Color insertion follows the same flow, with the selected color passed into the pure helper.

## Utility Changes
Extend `frontend/src/utils/markdownEditor.ts`.

Recommended API adjustment:

- add `text-color` to `MarkdownAction`, or add a dedicated `applyMarkdownColor(input, color)` helper.

The helper should stay pure and framework-free.

Validation:

- Accept hex colors such as `#ef4444`.
- Accept color strings produced by `tdesign-react` when practical.
- Reject or sanitize unsafe values such as strings containing quotes, semicolons, angle brackets, or `url(` before embedding in HTML.
- Fall back to a safe default color if the value is invalid.

## Error Handling
- Color insertion should never call APIs and should not need toast errors.
- Invalid color input should fall back to the last valid color or a default preset.
- Image upload behavior and failure toasts remain in `ArticleEditor`.
- Preview keeps the existing lazy import and suspense fallback.
- Empty content keeps the existing editor preview empty state pattern.

## Testing Plan
### Unit Tests
Extend `frontend/src/utils/markdownEditor.test.mjs` for:

- wrapping selected text in a color span,
- inserting a color span placeholder with selected placeholder text,
- rejecting unsafe color strings,
- preserving existing actions such as bold, heading, lists, code block, and link.

### Build and Lint
Run from `frontend/`:

- `npm run build`
- `npm run lint`

### Manual Verification
Start Vite and verify:

- normal editor still loads for new and edit article pages,
- preset color inserts the expected span,
- custom color inserts the expected span,
- color spans render in preview,
- edit, split, and preview modes still work,
- immersive writing mode expands the editor and can be exited,
- auto-save state still displays and content changes still trigger save behavior,
- image upload and pasted image upload still work,
- desktop and mobile widths have no toolbar or panel overlap,
- dark mode remains readable.

## Acceptance Criteria
- Authors can choose preset and custom font colors from a component-based color control.
- Color insertion produces safe inline HTML spans compatible with current Markdown rendering.
- The editor has an immersive writing mode with significantly larger writing space.
- `ArticleEditor` is simplified by delegating Markdown editor UI to a dedicated component.
- Existing article save, draft, auto-save, preview, and image upload behavior is preserved.
- Frontend build and lint pass.
