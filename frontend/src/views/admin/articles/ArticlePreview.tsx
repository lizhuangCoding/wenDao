import ReactMarkdown from 'react-markdown';
import rehypeHighlight from 'rehype-highlight';
import rehypeRaw from 'rehype-raw';
import remarkGfm from 'remark-gfm';
import 'highlight.js/styles/github-dark.css';
import { CollapsibleCodeBlock } from '@/components/article';

interface ArticlePreviewProps {
  content: string;
}

const getSourceLine = (node: any): number | undefined => {
  const line = node?.position?.start?.line;
  return typeof line === 'number' ? line : undefined;
};

const createAnnotatedBlock = (Tag: keyof JSX.IntrinsicElements) => {
  return ({ node, children, ...props }: any) => {
    const sourceLine = getSourceLine(node);
    return (
      <Tag {...props} data-md-line={sourceLine}>
        {children}
      </Tag>
    );
  };
};

const AnnotatedPre = ({ node, children, ...props }: any) => {
  const sourceLine = getSourceLine(node);

  return (
    <div data-md-line={sourceLine}>
      <CollapsibleCodeBlock {...props}>{children}</CollapsibleCodeBlock>
    </div>
  );
};

const components = {
  h1: createAnnotatedBlock('h1'),
  h2: createAnnotatedBlock('h2'),
  h3: createAnnotatedBlock('h3'),
  h4: createAnnotatedBlock('h4'),
  h5: createAnnotatedBlock('h5'),
  h6: createAnnotatedBlock('h6'),
  p: createAnnotatedBlock('p'),
  blockquote: createAnnotatedBlock('blockquote'),
  li: createAnnotatedBlock('li'),
  table: createAnnotatedBlock('table'),
  hr: createAnnotatedBlock('hr'),
  pre: AnnotatedPre,
};

export const ArticlePreview = ({ content }: ArticlePreviewProps) => (
  <ReactMarkdown
    remarkPlugins={[remarkGfm]}
    rehypePlugins={[rehypeHighlight, rehypeRaw]}
    components={components}
  >
    {content}
  </ReactMarkdown>
);
