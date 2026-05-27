export type MarkdownAction =
  | 'heading'
  | 'bold'
  | 'quote'
  | 'unordered-list'
  | 'ordered-list'
  | 'code-block'
  | 'inline-code'
  | 'link'
  | 'divider';

export interface ApplyMarkdownActionInput {
  text: string;
  selectionStart: number;
  selectionEnd: number;
  action: MarkdownAction;
}

export interface TextSelection {
  start: number;
  end: number;
}

export interface MarkdownTextEdit {
  start: number;
  end: number;
  replacement: string;
  selection: TextSelection;
}

export interface ApplyMarkdownActionResult {
  text: string;
  selection: TextSelection;
  edit: MarkdownTextEdit;
}

export type ApplyMarkdownTextInput = Omit<ApplyMarkdownActionInput, 'action'>;

export const DEFAULT_TEXT_COLOR = '#ef4444';

const HEX_COLOR_PATTERN = /^#(?:[0-9a-f]{3}|[0-9a-f]{6}|[0-9a-f]{8})$/i;

const replaceRange = (
  text: string,
  start: number,
  end: number,
  replacement: string,
  selection: TextSelection
): ApplyMarkdownActionResult => ({
  text: `${text.slice(0, start)}${replacement}${text.slice(end)}`,
  selection,
  edit: {
    start,
    end,
    replacement,
    selection,
  },
});

const getSelectedText = ({ text, selectionStart, selectionEnd }: ApplyMarkdownActionInput) =>
  text.slice(selectionStart, selectionEnd);

const getLineBounds = (text: string, selectionStart: number, selectionEnd: number) => {
  const lineStart = text.lastIndexOf('\n', Math.max(0, selectionStart - 1)) + 1;
  const nextBreak = text.indexOf('\n', selectionEnd);
  const lineEnd = nextBreak === -1 ? text.length : nextBreak;
  return { lineStart, lineEnd };
};

const prefixSelectedLines = (
  input: ApplyMarkdownActionInput,
  getPrefix: (index: number) => string
): ApplyMarkdownActionResult => {
  const { lineStart, lineEnd } = getLineBounds(input.text, input.selectionStart, input.selectionEnd);
  const selectedBlock = input.text.slice(lineStart, lineEnd);
  const lines = selectedBlock.split('\n');
  const replacement = lines.map((line, index) => `${getPrefix(index)}${line}`).join('\n');
  const prefixLength = getPrefix(0).length;

  return replaceRange(input.text, lineStart, lineEnd, replacement, {
    start: input.selectionStart + prefixLength,
    end: input.selectionEnd + prefixLength * lines.length,
  });
};

const wrapSelection = (
  input: ApplyMarkdownActionInput,
  before: string,
  after: string,
  fallback: string
): ApplyMarkdownActionResult => {
  const selectedText = getSelectedText(input) || fallback;
  const replacement = `${before}${selectedText}${after}`;
  const start = input.selectionStart + before.length;

  return replaceRange(input.text, input.selectionStart, input.selectionEnd, replacement, {
    start,
    end: start + selectedText.length,
  });
};

export const normalizeMarkdownColor = (
  color: string | undefined,
  fallback = DEFAULT_TEXT_COLOR
): string => {
  const trimmed = (color || '').trim();

  if (!HEX_COLOR_PATTERN.test(trimmed)) {
    return fallback;
  }

  const lower = trimmed.toLowerCase();
  if (lower.length === 4) {
    return `#${lower[1]}${lower[1]}${lower[2]}${lower[2]}${lower[3]}${lower[3]}`;
  }

  return lower;
};

export const applyMarkdownColor = (
  input: ApplyMarkdownTextInput,
  color: string
): ApplyMarkdownActionResult => {
  const safeColor = normalizeMarkdownColor(color);
  const selectedText = input.text.slice(input.selectionStart, input.selectionEnd) || '彩色文字';
  const before = `<span style="color: ${safeColor}">`;
  const after = '</span>';
  const replacement = `${before}${selectedText}${after}`;
  const start = input.selectionStart + before.length;

  return replaceRange(input.text, input.selectionStart, input.selectionEnd, replacement, {
    start,
    end: start + selectedText.length,
  });
};

export const applyMarkdownAction = (input: ApplyMarkdownActionInput): ApplyMarkdownActionResult => {
  switch (input.action) {
    case 'bold':
      return wrapSelection(input, '**', '**', '加粗文字');
    case 'inline-code':
      return wrapSelection(input, '`', '`', 'code');
    case 'heading': {
      const { lineStart, lineEnd } = getLineBounds(input.text, input.selectionStart, input.selectionEnd);
      const line = input.text.slice(lineStart, lineEnd);
      const replacement = line.startsWith('## ') ? line : `## ${line.replace(/^#{1,6}\s+/, '')}`;
      const delta = replacement.length - line.length;
      return replaceRange(input.text, lineStart, lineEnd, replacement, {
        start: input.selectionStart + Math.max(delta, 0),
        end: input.selectionEnd + Math.max(delta, 0),
      });
    }
    case 'quote':
      return prefixSelectedLines(input, () => '> ');
    case 'unordered-list':
      return prefixSelectedLines(input, () => '- ');
    case 'ordered-list':
      return prefixSelectedLines(input, (index) => `${index + 1}. `);
    case 'code-block': {
      const selectedText = getSelectedText(input);
      const replacement = `\`\`\`text\n${selectedText}\n\`\`\``;
      const cursor = input.selectionStart + '```text\n'.length;
      return replaceRange(input.text, input.selectionStart, input.selectionEnd, replacement, {
        start: cursor,
        end: selectedText ? cursor + selectedText.length : cursor,
      });
    }
    case 'link': {
      const selectedText = getSelectedText(input) || '链接文本';
      const replacement = `[${selectedText}](https://example.com)`;
      const start = input.selectionStart + 1;
      return replaceRange(input.text, input.selectionStart, input.selectionEnd, replacement, {
        start,
        end: start + selectedText.length,
      });
    }
    case 'divider': {
      const needsLeadingBreak = input.selectionStart > 0 && input.text[input.selectionStart - 1] !== '\n';
      const needsTrailingBreak = input.selectionEnd < input.text.length && input.text[input.selectionEnd] !== '\n';
      const replacement = `${needsLeadingBreak ? '\n' : ''}---${needsTrailingBreak ? '\n' : ''}`;
      const cursor = input.selectionStart + replacement.length;
      return replaceRange(input.text, input.selectionStart, input.selectionEnd, replacement, {
        start: cursor,
        end: cursor,
      });
    }
  }
};
