import ReactMarkdown from 'react-markdown';
import rehypeHighlight from 'rehype-highlight';
import rehypeRaw from 'rehype-raw';
import remarkGfm from 'remark-gfm';
import { slugify } from '@/utils/markdown';
import 'highlight.js/styles/github-dark.css';

interface ArticleContentProps {
  content: string;
}

// 递归提取 React 节点中的纯文本
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

// 为标题添加唯一 ID 的组件工厂
const createHeadingComponent = (level: number) => {
  return ({ children }: { children?: React.ReactNode }) => {
    const text = getTextContent(children);
    const id = slugify(text);

    const Tag = `h${level}` as keyof JSX.IntrinsicElements;
    const className = level === 1 ? 'text-3xl font-bold mt-8 mb-4 scroll-mt-24' :
                     level === 2 ? 'text-2xl font-bold mt-6 mb-3 scroll-mt-24' :
                     level === 3 ? 'text-xl font-bold mt-4 mb-2 scroll-mt-24' :
                     level === 4 ? 'text-lg font-bold mt-3 mb-2 scroll-mt-24' :
                     level === 5 ? 'text-base font-bold mt-2 mb-1 scroll-mt-24' :
                     'text-sm font-bold mt-2 mb-1 scroll-mt-24';

    return <Tag id={id} className={className}>{children}</Tag>;
  };
};

// 预定义标题组件，避免在渲染过程中动态创建
const headingComponents = {
  h1: createHeadingComponent(1),
  h2: createHeadingComponent(2),
  h3: createHeadingComponent(3),
  h4: createHeadingComponent(4),
  h5: createHeadingComponent(5),
  h6: createHeadingComponent(6),
};

export const ArticleContent = ({ content }: ArticleContentProps) => {
  return (
    <div className="markdown-body">
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        rehypePlugins={[rehypeHighlight, rehypeRaw]}
        components={headingComponents}
      >
        {content}
      </ReactMarkdown>
    </div>
  );
};
