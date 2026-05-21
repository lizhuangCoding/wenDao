import type { ChatArticleReference, ChatReferenceGroups } from '@/types';

export const emptyReferenceGroups = (): ChatReferenceGroups => ({ blog: [], external: [] });

const parseReferenceLinks = (block: string): ChatArticleReference[] => {
  const references: ChatArticleReference[] = [];
  const seen = new Set<string>();
  const linkPattern = /-\s*\[([^\]]+)\]\(([^)]+)\)/g;
  let match: RegExpExecArray | null;

  while ((match = linkPattern.exec(block)) !== null) {
    const title = match[1].trim();
    const url = match[2].trim();
    const key = `${title}|${url}`;
    if (!title || !url || seen.has(key)) continue;
    seen.add(key);
    references.push({ title, url });
  }

  return references;
};

export const parseChatArticleReferences = (content: string): { body: string; references: ChatReferenceGroups } => {
  const markerPattern = /\n{0,2}(参考博主文章|参考外部文章|参考文章)\s*\n/g;
  const markerMatch = markerPattern.exec(content);
  if (!markerMatch || markerMatch.index === undefined) {
    return { body: content, references: emptyReferenceGroups() };
  }

  const body = content.slice(0, markerMatch.index).trimEnd();
  const references = emptyReferenceGroups();
  const markers: Array<{ title: string; start: number; contentStart: number }> = [];

  markers.push({
    title: markerMatch[1],
    start: markerMatch.index,
    contentStart: markerMatch.index + markerMatch[0].length,
  });

  let match: RegExpExecArray | null;
  while ((match = markerPattern.exec(content)) !== null) {
    markers.push({ title: match[1], start: match.index, contentStart: match.index + match[0].length });
  }

  markers.forEach((marker, index) => {
    const nextStart = markers[index + 1]?.start ?? content.length;
    const links = parseReferenceLinks(content.slice(marker.contentStart, nextStart));
    if (marker.title === '参考外部文章') {
      references.external.push(...links);
    } else {
      references.blog.push(...links);
    }
  });

  if (references.blog.length === 0 && references.external.length === 0) {
    return { body: content, references: emptyReferenceGroups() };
  }

  return { body, references };
};
