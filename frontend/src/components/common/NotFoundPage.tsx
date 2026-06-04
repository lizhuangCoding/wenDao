import { Link } from 'react-router-dom';
import { Layout } from './Layout';

export const NotFoundPage = () => (
  <Layout>
    <div className="mx-auto flex min-h-[60vh] max-w-lg flex-col items-center justify-center px-6 py-24 text-center">
      <p className="text-xs font-black uppercase tracking-[0.3em] text-primary-600 dark:text-primary-400">
        404
      </p>
      <h1 className="mt-4 text-4xl font-serif font-black text-neutral-900 dark:text-neutral-100">
        页面不存在
      </h1>
      <p className="mt-4 text-sm font-medium leading-6 text-neutral-500 dark:text-neutral-400">
        这个地址没有可访问的页面，可能是链接已经变更，或者你输入了错误的路径。
      </p>
      <Link to="/" className="btn btn-primary mt-8">
        返回首页
      </Link>
    </div>
  </Layout>
);

