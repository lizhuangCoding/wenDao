export interface TocItem {
  id: string;
  text: string;
  level: number;
}

const getVisibleHeadingText = (text: string): string => {
  return text
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replace(/<[^>]*>/g, '')
    .replace(/[*_`~]/g, '')
    .trim();
};

export const markdownToPlainText = (content: string): string => {
  if (!content) return '';

  return content
    .replace(/\r\n?/g, '\n')
    .replace(/```[\s\S]*?```/g, (block) =>
      block.replace(/^```[^\n]*\n?/, '').replace(/\n?```$/, '\n')
    )
    .replace(/~~~[\s\S]*?~~~/g, (block) =>
      block.replace(/^~~~[^\n]*\n?/, '').replace(/\n?~~~$/, '\n')
    )
    .replace(/!\[([^\]]*)\]\([^)]+\)/g, '$1')
    .replace(/\[([^\]]+)\]\([^)]+\)/g, '$1')
    .replace(/\[([^\]]+)\]\[[^\]]*\]/g, '$1')
    .replace(/^\s{0,3}\[[^\]]+\]:\s+\S+.*$/gm, '')
    .replace(/<br\s*\/?>/gi, '\n')
    .replace(/<[^>]*>/g, '')
    .replace(/^\s*\|?[\s:-]+\|[\s|:-]*$/gm, '')
    .split('\n')
    .map((line) =>
      line
        .replace(/^\s{0,3}#{1,6}\s+/, '')
        .replace(/^\s{0,3}>\s?/, '')
        .replace(/^\s*([-*+]|\d+[.)])\s+/, '')
        .replace(/^\s*\[[ xX]\]\s+/, '')
        .replace(/^\s*[-*_]{3,}\s*$/, '')
    )
    .join('\n')
    .replace(/`([^`]+)`/g, '$1')
    .replace(/\*\*([^*]+)\*\*/g, '$1')
    .replace(/__([^_]+)__/g, '$1')
    .replace(/\*([^*]+)\*/g, '$1')
    .replace(/_([^_]+)_/g, '$1')
    .replace(/~~([^~]+)~~/g, '$1')
    .replace(/\|/g, ' ')
    .replace(/\s+/g, ' ')
    .trim();
};

const countProseUnits = (content: string): number => {
  const normalized = markdownToPlainText(content);
  if (!normalized) return 0;

  const chineseCharacters = normalized.match(/[\u4e00-\u9fff]/g)?.length ?? 0;
  const englishWords = normalized.match(/[A-Za-z0-9]+(?:'[A-Za-z0-9]+)?/g)?.length ?? 0;

  return (chineseCharacters / 350) + (englishWords / 220);
};

const countCodeUnits = (code: string): number => {
  const normalized = code.trimEnd();
  if (!normalized) return 0;

  const lines = normalized.split(/\r?\n/).length;
  const characters = normalized.replace(/\s+/g, '').length;

  return Math.max(lines / 4, characters / 120);
};

const stripCodeBlocks = (content: string): string => {
  return content
    .replace(/```[^\n]*\n[\s\S]*?```/g, '\n')
    .replace(/~~~[^\n]*\n[\s\S]*?~~~/g, '\n');
};

const extractCodeBlocks = (content: string): string[] => {
  const blocks: string[] = [];

  content.replace(/```[^\n]*\n([\s\S]*?)```/g, (_, code: string) => {
    blocks.push(code);
    return '\n';
  });

  content.replace(/~~~[^\n]*\n([\s\S]*?)~~~/g, (_, code: string) => {
    blocks.push(code);
    return '\n';
  });

  return blocks;
};

export const estimateReadingTime = (content: string): number => {
  if (!content?.trim()) return 1;

  const proseUnits = countProseUnits(stripCodeBlocks(content));
  const codeUnits = extractCodeBlocks(content).reduce((total, block) => total + countCodeUnits(block), 0);
  const totalMinutes = proseUnits + codeUnits;

  return Math.max(1, Math.ceil(totalMinutes));
};

/**
 * 稳定的 Slug 生成器
 * 注意：由于 extractHeadings 和 ReactMarkdown 渲染是分开的，
 * 我们需要一种不依赖外部状态也能生成一致 ID 的方式。
 * 如果文章中有完全重复的标题，建议用户微调标题内容。
 */
export const slugify = (text: string): string => {
  return getVisibleHeadingText(text)
    .toLowerCase()
    .trim()
    // 移除特殊字符
    .replace(/[^\w\s\u4e00-\u9fa5-]/g, '')
    // 空格和下划线转为连字符
    .replace(/[\s_-]+/g, '-')
    // 移除首尾连字符
    .replace(/^-+|-+$/g, '');
};

/**
 * 提取文章目录
 * 逻辑：
 * 1. 过滤掉代码块中的内容，防止误判
 * 2. 匹配标准 Markdown 标题
 * 3. 生成 ID 时清理 Markdown 语法
 */
export const extractHeadings = (content: string): TocItem[] => {
  if (!content) return [];

  // 1. 移除代码块，防止匹配到代码块内的 # 符号
  const contentWithoutCode = content.replace(/```[\s\S]*?```/g, '');

  // 2. 匹配标题 (支持 1-6 级，兼容有无空格)
  // ^#{1,6}\s*(.+?)$
  const headingRegex = /^#{1,6}\s+(.+?)$/gm;
  const headings: TocItem[] = [];
  let match;

  while ((match = headingRegex.exec(contentWithoutCode)) !== null) {
    const rawLine = match[0];
    const level = rawLine.match(/^#+/)?.[0].length || 1;
    const text = match[1].trim();

    const plainText = getVisibleHeadingText(text);

    headings.push({
      id: slugify(plainText),
      text: plainText,
      level,
    });
  }

  return headings;
};
