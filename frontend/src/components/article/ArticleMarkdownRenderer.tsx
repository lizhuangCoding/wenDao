import ReactMarkdown from 'react-markdown';
import rehypeHighlight from 'rehype-highlight';
import remarkGfm from 'remark-gfm';
import { slugify } from '@/utils/markdown';
import { CollapsibleCodeBlock } from './CollapsibleCodeBlock';
import 'highlight.js/styles/github-dark.css';

interface ArticleMarkdownRendererProps {
  content: string;
}

const getTextContent = (node: React.ReactNode): string => {
  if (typeof node === 'string') return node;
  if (typeof node === 'number') return String(node);
  if (!node) return '';
  if (Array.isArray(node)) return node.map(getTextContent).join('');
  if (typeof node === 'object' && 'props' in node) {
    return getTextContent((node as React.ReactElement).props.children);
  }
  return '';
};

const createHeadingComponent = (level: number) => {
  return ({ children }: { children?: React.ReactNode }) => {
    const text = getTextContent(children);
    const id = slugify(text);

    const Tag = `h${level}` as React.ElementType;
    const className = 'scroll-mt-24';

    return <Tag id={id} className={className}>{children}</Tag>;
  };
};

const headingComponents = {
  h1: createHeadingComponent(1),
  h2: createHeadingComponent(2),
  h3: createHeadingComponent(3),
  h4: createHeadingComponent(4),
  h5: createHeadingComponent(5),
  h6: createHeadingComponent(6),
};

export const ArticleMarkdownRenderer = ({ content }: ArticleMarkdownRendererProps) => (
  <div className="markdown-body">
    <ReactMarkdown
      remarkPlugins={[remarkGfm]}
      rehypePlugins={[rehypeHighlight]}
      components={{ ...headingComponents, pre: CollapsibleCodeBlock }}
    >
      {content}
    </ReactMarkdown>
  </div>
);
