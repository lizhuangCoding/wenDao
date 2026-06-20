import { FormEvent, useEffect, useMemo, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { useQuery } from '@tanstack/react-query';
import { Search as SearchIcon } from 'lucide-react';
import { categoryApi, searchApi, tagApi } from '@/api';
import {
  Button,
  EmptyState,
  ErrorState,
  Layout,
  Loading,
  PageHeader,
  PageShell,
  Pagination,
  Panel,
  SelectInput,
  TextInput,
} from '@/components/common';
import { formatDate } from '@/utils';

const parsePositiveInt = (value: string | null) => {
  if (!value) return undefined;
  const parsed = Number(value);
  return Number.isFinite(parsed) && parsed > 0 ? parsed : undefined;
};

const SEARCH_HISTORY_KEY = 'wendao-search-history';
const MAX_SEARCH_HISTORY = 8;

const readSearchHistory = () => {
  if (typeof window === 'undefined') return [];
  try {
    const parsed = JSON.parse(window.localStorage.getItem(SEARCH_HISTORY_KEY) || '[]');
    return Array.isArray(parsed) ? parsed.filter((item): item is string => typeof item === 'string') : [];
  } catch {
    return [];
  }
};

const saveSearchHistory = (term: string) => {
  const keyword = term.trim();
  if (!keyword || typeof window === 'undefined') return readSearchHistory();

  const previous = readSearchHistory();
  const next = [
    keyword,
    ...previous.filter((item) => item.trim().toLowerCase() !== keyword.toLowerCase()),
  ].slice(0, MAX_SEARCH_HISTORY);

  try {
    window.localStorage.setItem(SEARCH_HISTORY_KEY, JSON.stringify(next));
  } catch {
    // Ignore private-mode storage failures; search itself should still work.
  }
  return next;
};

export const Search = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const query = searchParams.get('q')?.trim() ?? '';
  const categoryID = parsePositiveInt(searchParams.get('category_id'));
  const tagID = parsePositiveInt(searchParams.get('tag_id'));
  const currentPage = parsePositiveInt(searchParams.get('page')) ?? 1;
  const [inputValue, setInputValue] = useState(query);
  const [searchHistory, setSearchHistory] = useState<string[]>(() => readSearchHistory());

  const hasSearchCriteria = query !== '' || !!categoryID || !!tagID;

  const { data: categories } = useQuery({
    queryKey: ['categories'],
    queryFn: categoryApi.getCategories,
  });

  const { data: tags } = useQuery({
    queryKey: ['tags'],
    queryFn: tagApi.getTags,
  });

  const {
    data: searchData,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery({
    queryKey: ['site-search', query, categoryID, tagID, currentPage],
    queryFn: () =>
      searchApi.searchArticles({
        q: query || undefined,
        category_id: categoryID,
        tag_id: tagID,
        page: currentPage,
        pageSize: 10,
      }),
    enabled: hasSearchCriteria,
    placeholderData: (previousData) => previousData,
  });

  const totalPages = Math.max(1, searchData?.totalPages ?? 1);
  const resultSummary = useMemo(() => {
    if (!hasSearchCriteria) return '输入关键词，或选择分类/标签开始搜索';
    if (!searchData) return '正在搜索';
    return `找到 ${searchData.total} 篇相关文章`;
  }, [hasSearchCriteria, searchData]);

  const updateParams = (next: { q?: string; category_id?: number; tag_id?: number; page?: number }) => {
    const params = new URLSearchParams(searchParams);
    const writeString = (key: string, value?: string) => {
      if (value && value.trim()) {
        params.set(key, value.trim());
      } else {
        params.delete(key);
      }
    };
    const writeNumber = (key: string, value?: number) => {
      if (value && value > 0) {
        params.set(key, String(value));
      } else {
        params.delete(key);
      }
    };

    if ('q' in next) writeString('q', next.q);
    if ('category_id' in next) writeNumber('category_id', next.category_id);
    if ('tag_id' in next) writeNumber('tag_id', next.tag_id);
    if ('page' in next) writeNumber('page', next.page && next.page > 1 ? next.page : undefined);
    setSearchParams(params);
  };

  useEffect(() => {
    setInputValue(query);
    if (query) {
      setSearchHistory(saveSearchHistory(query));
    }
  }, [query]);

  const runKeywordSearch = (keyword: string) => {
    const nextKeyword = keyword.trim();
    setInputValue(nextKeyword);
    if (nextKeyword) {
      setSearchHistory(saveSearchHistory(nextKeyword));
    }
    updateParams({ q: nextKeyword, page: 1 });
  };

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault();
    runKeywordSearch(inputValue);
  };

  return (
    <Layout>
      <PageShell width="default" padding="lg">
        <PageHeader
          eyebrow="Site Search"
          title="站内搜索"
          description="搜索文章标题、摘要、正文、分类和标签。"
          className="mb-8"
        />

        <Panel className="space-y-4">
          <form onSubmit={handleSubmit} className="grid gap-3 md:grid-cols-[1fr_auto]">
            <TextInput
              value={inputValue}
              onChange={(event) => setInputValue(event.target.value)}
              placeholder="搜索文章、主题或关键词"
              leading={<SearchIcon className="h-4 w-4" />}
            />
            <Button type="submit">搜索</Button>
          </form>

          <div className="grid gap-3 sm:grid-cols-2">
            <SelectInput
              value={categoryID ? String(categoryID) : ''}
              onChange={(event) =>
                updateParams({
                  category_id: parsePositiveInt(event.target.value),
                  page: 1,
                })
              }
            >
              <option value="">全部分类</option>
              {categories?.map((category) => (
                <option key={category.id} value={category.id}>
                  {category.name}
                </option>
              ))}
            </SelectInput>
            <SelectInput
              value={tagID ? String(tagID) : ''}
              onChange={(event) =>
                updateParams({
                  tag_id: parsePositiveInt(event.target.value),
                  page: 1,
                })
              }
            >
              <option value="">全部标签</option>
              {tags?.map((tag) => (
                <option key={tag.id} value={tag.id}>
                  {tag.name}
                </option>
              ))}
            </SelectInput>
          </div>

          {!hasSearchCriteria && (
            <section className="border-t border-neutral-200 pt-4 dark:border-neutral-700">
              <div className="mb-3 flex items-center justify-between gap-3">
                <h2 className="text-sm font-black text-neutral-800 dark:text-neutral-100">搜索历史</h2>
                {searchHistory.length > 0 && (
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => {
                      window.localStorage.removeItem(SEARCH_HISTORY_KEY);
                      setSearchHistory([]);
                    }}
                  >
                    清空历史
                  </Button>
                )}
              </div>
              {searchHistory.length > 0 ? (
                <div className="flex flex-wrap gap-2">
                  {searchHistory.map((term) => (
                    <button
                      key={term}
                      type="button"
                      onClick={() => runKeywordSearch(term)}
                      className="rounded-full border border-neutral-200 px-3 py-1.5 text-xs font-bold text-neutral-600 transition-colors hover:border-primary-200 hover:bg-primary-50 hover:text-primary-700 dark:border-neutral-700 dark:text-neutral-300 dark:hover:border-primary-700 dark:hover:bg-primary-900/20 dark:hover:text-primary-300"
                    >
                      {term}
                    </button>
                  ))}
                </div>
              ) : (
                <p className="text-sm font-medium text-neutral-400 dark:text-neutral-500">
                  暂无搜索历史
                </p>
              )}
            </section>
          )}
        </Panel>

        <div className="mt-8 flex items-center justify-between gap-4 border-b border-neutral-200 pb-4 dark:border-neutral-700">
          <p className="text-sm font-bold text-neutral-500 dark:text-neutral-400">{resultSummary}</p>
          {hasSearchCriteria && (
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => {
                setInputValue('');
                setSearchParams(new URLSearchParams());
              }}
            >
              清空
            </Button>
          )}
        </div>

        {!hasSearchCriteria ? (
          <EmptyState title="开始一次站内搜索" description="输入关键词，或者选择分类/标签查看相关文章。" className="py-24" />
        ) : isLoading ? (
          <div className="flex justify-center py-20">
            <Loading />
          </div>
        ) : isError ? (
          <ErrorState message={(error as any)?.message || '搜索失败'} onRetry={() => refetch()} className="mt-10" />
        ) : searchData?.data?.length === 0 ? (
          <EmptyState title="没有找到相关文章" description="换一个关键词，或放宽分类/标签筛选。" className="py-24" />
        ) : (
          <div className="divide-y divide-neutral-200 dark:divide-neutral-700">
            {searchData?.data.map((result) => {
              const article = result.article;
              return (
                <article key={article.id} className="py-7">
                  <div className="mb-3 flex flex-wrap items-center gap-2 text-xs font-semibold text-neutral-400">
                    <span className="rounded-full bg-primary-50 px-2.5 py-1 text-primary-600 dark:bg-primary-500/10 dark:text-primary-300">
                      {article.category?.name}
                    </span>
                    <span>{formatDate(article.created_at)}</span>
                    {result.matched_fields.map((field) => (
                      <span key={field} className="rounded-full border border-neutral-200 px-2 py-1 dark:border-neutral-700">
                        {field}
                      </span>
                    ))}
                  </div>
                  <Link to={`/article/${article.slug}`} className="group block">
                    <h2 className="text-2xl font-black leading-tight text-neutral-900 transition-colors group-hover:text-primary-600 dark:text-neutral-100 dark:group-hover:text-primary-400">
                      {article.title}
                    </h2>
                    <p
                      className="mt-3 text-sm leading-7 text-neutral-500 [&_mark]:rounded [&_mark]:bg-yellow-200 [&_mark]:px-1 [&_mark]:text-neutral-950 dark:text-neutral-400 dark:[&_mark]:bg-yellow-300"
                      dangerouslySetInnerHTML={{ __html: result.snippet }}
                    />
                  </Link>
                  {article.tags && article.tags.length > 0 && (
                    <div className="mt-4 flex flex-wrap gap-2">
                      {article.tags.map((tag) => (
                        <span
                          key={tag.id}
                          className="rounded-full border border-neutral-200 px-2.5 py-1 text-xs font-bold text-neutral-500 dark:border-neutral-700 dark:text-neutral-400"
                        >
                          #{tag.name}
                        </span>
                      ))}
                    </div>
                  )}
                </article>
              );
            })}
          </div>
        )}

        {searchData && searchData.totalPages > 1 && (
          <Pagination
            page={currentPage}
            totalPages={totalPages}
            total={searchData.total}
            pageSize={10}
            onChange={(page) => updateParams({ page })}
            previousLabel="上一页"
            nextLabel="下一页"
            className="mt-10 border-t border-neutral-200 pt-8 dark:border-neutral-700"
          />
        )}
      </PageShell>
    </Layout>
  );
};
