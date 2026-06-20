type ExportableArticle = {
  title: string;
  summary?: string;
  content: string;
  category?: { name?: string };
  author?: { username?: string };
  tags?: Array<{ name: string }>;
  published_at?: string;
  created_at?: string;
};

const formatExportDate = (value?: string) => {
  if (!value) return '';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleDateString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  });
};

export const getArticleMarkdownFilename = (article: ExportableArticle) => {
  const safeTitle = article.title.trim().replace(/[\\/:*?"<>|]/g, '_') || 'article';
  return `${safeTitle}.md`;
};

export const buildArticleMarkdown = (article: ExportableArticle) => {
  const lines: string[] = [`# ${article.title}`, ''];

  if (article.summary?.trim()) {
    lines.push(`> ${article.summary.trim()}`, '');
  }

  const metadata = [
    article.category?.name ? `- 分类：${article.category.name}` : '',
    article.tags && article.tags.length > 0 ? `- 标签：${article.tags.map((tag) => tag.name).join(', ')}` : '',
    article.author?.username ? `- 作者：${article.author.username}` : '',
    article.published_at || article.created_at ? `- 发布时间：${formatExportDate(article.published_at || article.created_at)}` : '',
  ].filter(Boolean);

  if (metadata.length > 0) {
    lines.push(...metadata, '');
  }

  lines.push('---', '', article.content.trim(), '');
  return lines.join('\n');
};

export const downloadArticleMarkdown = (article: ExportableArticle) => {
  const blob = new Blob([buildArticleMarkdown(article)], { type: 'text/markdown;charset=utf-8' });
  const url = window.URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = getArticleMarkdownFilename(article);
  document.body.appendChild(anchor);
  anchor.click();
  window.URL.revokeObjectURL(url);
  document.body.removeChild(anchor);
};
