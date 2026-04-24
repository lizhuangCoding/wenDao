import { useState } from 'react';
import { Link } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { knowledgeDocumentApi } from '@/api/knowledgeDocument';

export const KnowledgeDocumentList = () => {
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState<'pending_review' | 'approved' | 'rejected' | ''>('');
  const { data, isLoading } = useQuery({
    queryKey: ['admin-knowledge-documents', page, status],
    queryFn: () => knowledgeDocumentApi.getKnowledgeDocuments({ page, pageSize: 10, status }),
  });

  if (isLoading) {
    return <div className="p-6 text-sm text-neutral-500">加载中...</div>;
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between gap-4">
        <h1 className="text-3xl font-serif font-bold text-neutral-800 dark:text-neutral-100">知识文档审核</h1>
        <div className="flex gap-2">
          {[
            ['', '全部'],
            ['pending_review', '待审核'],
            ['approved', '已通过'],
            ['rejected', '已拒绝'],
          ].map(([value, label]) => (
            <button
              key={value}
              onClick={() => {
                setStatus(value as typeof status);
                setPage(1);
              }}
              className={`rounded-xl px-3 py-2 text-xs font-bold ${
                status === value ? 'bg-primary-600 text-white' : 'bg-neutral-100 text-neutral-600 dark:bg-neutral-800 dark:text-neutral-300'
              }`}
            >
              {label}
            </button>
          ))}
        </div>
      </div>
      <div className="bg-white dark:bg-neutral-900 rounded-2xl shadow-sm border border-neutral-100 dark:border-neutral-800 overflow-hidden">
        <table className="w-full text-left border-collapse">
          <thead>
            <tr>
              <th className="px-6 py-4">标题</th>
              <th className="px-6 py-4">状态</th>
              <th className="px-6 py-4">创建时间</th>
              <th className="px-6 py-4 text-right">操作</th>
            </tr>
          </thead>
          <tbody>
            {data?.data?.map((doc) => (
              <tr key={doc.id} className="border-t border-neutral-100 dark:border-neutral-800">
                <td className="px-6 py-4">{doc.title}</td>
                <td className="px-6 py-4">
                  <div className="flex flex-col gap-1">
                    <span>{doc.status}</span>
                    {doc.article_id && <span className="text-xs text-primary-600">已生成文章 #{doc.article_id}</span>}
                  </div>
                </td>
                <td className="px-6 py-4">{new Date(doc.created_at).toLocaleString()}</td>
                <td className="px-6 py-4 text-right">
                  <Link to={`/admin/knowledge-documents/${doc.id}`} className="text-primary-600 hover:underline">
                    查看
                  </Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
};
