import { getSynchronizedScrollTop } from '@/utils/markdownEditor';

const parseAnchorLine = (element: Element) => {
  const line = Number.parseInt(element.getAttribute('data-md-line') || '', 10);
  return Number.isFinite(line) ? line : undefined;
};

const getPreviewAnchors = (preview: HTMLElement) => {
  const previewRect = preview.getBoundingClientRect();
  const anchorsByLine = new Map<number, number>();

  Array.from(preview.querySelectorAll<HTMLElement>('[data-md-line]')).forEach((element) => {
    const line = parseAnchorLine(element);
    if (typeof line !== 'number') return;

    const top = element.getBoundingClientRect().top - previewRect.top + preview.scrollTop;
    const currentTop = anchorsByLine.get(line);
    anchorsByLine.set(line, currentTop === undefined ? top : Math.min(currentTop, top));
  });

  return Array.from(anchorsByLine.entries())
    .map(([line, top]) => ({ line, top }))
    .sort((a, b) => a.line - b.line);
};

const findLastAnchor = <T,>(items: T[], predicate: (item: T) => boolean) => {
  for (let index = items.length - 1; index >= 0; index -= 1) {
    if (predicate(items[index])) return items[index];
  }
  return undefined;
};

const clampScrollTop = (scrollTop: number, target: HTMLElement) => {
  const maxScrollTop = Math.max(0, target.scrollHeight - target.clientHeight);
  return Math.min(maxScrollTop, Math.max(0, scrollTop));
};

const escapeHtml = (text: string) => {
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
};

const getMarkdownScrollAnchorLines = (content: string) => {
  const lines = content.split(/\r\n|\r|\n/);
  const anchors = new Set<number>([1]);
  let isInFence = false;

  lines.forEach((line, index) => {
    const lineNumber = index + 1;
    const trimmed = line.trim();
    const previousTrimmed = lines[index - 1]?.trim() ?? '';

    if (/^(```|~~~)/.test(trimmed)) {
      anchors.add(lineNumber);
      isInFence = !isInFence;
      return;
    }

    if (isInFence || !trimmed) return;

    if (
      /^#{1,6}\s+/.test(trimmed) ||
      /^([-*+]|\d+[.)])\s+/.test(trimmed) ||
      /^>\s?/.test(trimmed) ||
      /^!\[[^\]]*]\([^)]+\)/.test(trimmed) ||
      /^([-*_]\s*){3,}$/.test(trimmed) ||
      /^\|.*\|$/.test(trimmed) ||
      previousTrimmed === ''
    ) {
      anchors.add(lineNumber);
    }
  });

  anchors.add(lines.length);
  return Array.from(anchors).sort((a, b) => a - b);
};

const syncEditorMirrorStyles = (textarea: HTMLTextAreaElement, mirror: HTMLDivElement) => {
  const styles = window.getComputedStyle(textarea);

  mirror.style.position = 'absolute';
  mirror.style.left = '-10000px';
  mirror.style.top = '0';
  mirror.style.visibility = 'hidden';
  mirror.style.pointerEvents = 'none';
  mirror.style.width = `${textarea.clientWidth}px`;
  mirror.style.boxSizing = styles.boxSizing;
  mirror.style.padding = styles.padding;
  mirror.style.border = styles.border;
  mirror.style.font = styles.font;
  mirror.style.lineHeight = styles.lineHeight;
  mirror.style.letterSpacing = styles.letterSpacing;
  mirror.style.whiteSpace = 'pre-wrap';
  mirror.style.overflowWrap = 'break-word';
  mirror.style.tabSize = styles.getPropertyValue('tab-size');
};

const getEditorMarkdownAnchors = (
  textarea: HTMLTextAreaElement,
  mirror: HTMLDivElement,
  content: string
) => {
  const anchorLines = new Set(getMarkdownScrollAnchorLines(content));
  const html = content
    .split(/\r\n|\r|\n/)
    .map((line, index) => {
      const lineNumber = index + 1;
      const marker = anchorLines.has(lineNumber)
        ? `<span data-editor-md-line="${lineNumber}"></span>`
        : '';
      return `${marker}${escapeHtml(line)}`;
    })
    .join('\n');

  syncEditorMirrorStyles(textarea, mirror);
  mirror.innerHTML = html || '<span data-editor-md-line="1"></span>';
  const mirrorRect = mirror.getBoundingClientRect();

  return Array.from(mirror.querySelectorAll<HTMLElement>('[data-editor-md-line]'))
    .map((element) => ({
      line: Number.parseInt(element.getAttribute('data-editor-md-line') || '', 10),
      top: element.getBoundingClientRect().top - mirrorRect.top,
    }))
    .filter((item) => Number.isFinite(item.line))
    .sort((a, b) => a.top - b.top);
};

const getInterpolatedTopForLine = (
  anchors: Array<{ line: number; top: number }>,
  sourceLine: number
) => {
  const previousAnchor = findLastAnchor(anchors, (item) => item.line <= sourceLine) ?? anchors[0];
  const nextAnchor = anchors.find((item) => item.line > sourceLine);

  if (!nextAnchor || nextAnchor.line === previousAnchor.line) {
    return previousAnchor.top;
  }

  const progress = Math.min(
    1,
    Math.max(0, (sourceLine - previousAnchor.line) / (nextAnchor.line - previousAnchor.line))
  );

  return previousAnchor.top + progress * (nextAnchor.top - previousAnchor.top);
};

const getScrollMap = (
  editor: HTMLTextAreaElement,
  preview: HTMLElement,
  content: string,
  editorMirror: HTMLDivElement
) => {
  const editorAnchors = getEditorMarkdownAnchors(editor, editorMirror, content);
  const previewAnchors = getPreviewAnchors(preview);
  if (editorAnchors.length === 0 || previewAnchors.length === 0) return [];

  const editorMaxScrollTop = Math.max(0, editor.scrollHeight - editor.clientHeight);
  const previewMaxScrollTop = Math.max(0, preview.scrollHeight - preview.clientHeight);
  const map = editorAnchors.map((editorAnchor) => ({
    editorTop: clampScrollTop(editorAnchor.top, editor),
    previewTop: clampScrollTop(
      getInterpolatedTopForLine(previewAnchors, editorAnchor.line) - 16,
      preview
    ),
  }));

  map.unshift({ editorTop: 0, previewTop: 0 });
  map.push({ editorTop: editorMaxScrollTop, previewTop: previewMaxScrollTop });

  return map
    .sort((a, b) => a.editorTop - b.editorTop)
    .filter((item, index, items) => index === 0 || item.editorTop !== items[index - 1].editorTop);
};

const interpolateScrollMap = (
  scrollTop: number,
  map: Array<{ sourceTop: number; targetTop: number }>
) => {
  if (map.length === 0) return undefined;

  const previousPoint = findLastAnchor(map, (item) => item.sourceTop <= scrollTop) ?? map[0];
  const nextPoint = map.find((item) => item.sourceTop > scrollTop);

  if (!nextPoint || nextPoint.sourceTop === previousPoint.sourceTop) {
    return previousPoint.targetTop;
  }

  const progress = Math.min(
    1,
    Math.max(0, (scrollTop - previousPoint.sourceTop) / (nextPoint.sourceTop - previousPoint.sourceTop))
  );

  return previousPoint.targetTop + progress * (nextPoint.targetTop - previousPoint.targetTop);
};

export const getSyncedPreviewScrollTop = (
  editor: HTMLTextAreaElement,
  preview: HTMLElement,
  content: string,
  editorMirror: HTMLDivElement
) => {
  const anchorScrollTop = interpolateScrollMap(
    editor.scrollTop,
    getScrollMap(editor, preview, content, editorMirror).map((item) => ({
      sourceTop: item.editorTop,
      targetTop: item.previewTop,
    }))
  );

  return (
    anchorScrollTop ??
    getSynchronizedScrollTop({
      sourceScrollTop: editor.scrollTop,
      sourceScrollHeight: editor.scrollHeight,
      sourceClientHeight: editor.clientHeight,
      targetScrollHeight: preview.scrollHeight,
      targetClientHeight: preview.clientHeight,
    })
  );
};

export const getSyncedEditorScrollTop = (
  preview: HTMLElement,
  editor: HTMLTextAreaElement,
  content: string,
  editorMirror: HTMLDivElement
) => {
  const anchorScrollTop = interpolateScrollMap(
    preview.scrollTop,
    getScrollMap(editor, preview, content, editorMirror).map((item) => ({
      sourceTop: item.previewTop,
      targetTop: item.editorTop,
    }))
  );

  return (
    anchorScrollTop ??
    getSynchronizedScrollTop({
      sourceScrollTop: preview.scrollTop,
      sourceScrollHeight: preview.scrollHeight,
      sourceClientHeight: preview.clientHeight,
      targetScrollHeight: editor.scrollHeight,
      targetClientHeight: editor.clientHeight,
    })
  );
};
