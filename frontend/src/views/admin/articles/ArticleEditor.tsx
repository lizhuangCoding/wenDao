import {
  type ChangeEvent,
  type ClipboardEvent,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation } from '@tanstack/react-query';
import dayjs from 'dayjs';
import { DatePicker, Select } from 'tdesign-react';
import type { DatePickerProps } from 'tdesign-react';
import { CalendarClock } from 'lucide-react';
import { articleApi, categoryApi, uploadApi, chatApi } from '@/api';
import { Loading, ErrorState } from '@/components/common';
import { useUIStore } from '@/store';
import { getArticlePrimaryActionLabel } from '@/utils/pageBehavior';
import { MarkdownWritingStudio } from './components/MarkdownWritingStudio';
import 'tdesign-react/es/style/index.css';

const SCHEDULE_PICKER_FORMAT = 'YYYY-MM-DD HH:mm';
type ScheduledPickerValue = Parameters<NonNullable<DatePickerProps['onChange']>>[0];

const getSingleScheduledPickerValue = (value: ScheduledPickerValue) => {
  if (!value || Array.isArray(value)) return '';
  return value instanceof Date ? dayjs(value).format(SCHEDULE_PICKER_FORMAT) : String(value);
};

const toScheduledPickerValue = (value: string) => {
  if (!value) return '';
  const parsed = dayjs(value);
  return parsed.isValid() ? parsed.format(SCHEDULE_PICKER_FORMAT) : '';
};

const toScheduledPayloadValue = (value: ScheduledPickerValue) => {
  const pickerValue = getSingleScheduledPickerValue(value);
  if (!pickerValue) return '';

  const parsed = dayjs(pickerValue);
  return parsed.isValid() ? parsed.second(0).millisecond(0).toDate().toISOString() : '';
};

export const ArticleEditor = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { showToast } = useUIStore();
  const isEdit = !!id;
  const contentInputRef = useRef<HTMLTextAreaElement>(null);
  const contentImageInputRef = useRef<HTMLInputElement>(null);
  const [isGeneratingSummary, setIsGeneratingSummary] = useState(false);
  const [lastSavedTime, setLastSavedTime] = useState<string | null>(null);
  const [isAutoSaving, setIsAutoSaving] = useState(false);
  const [isWritingFocused, setIsWritingFocused] = useState(false);
  const lastSavedDataRef = useRef({ title: '', content: '', summary: '' });

  const [formData, setFormData] = useState({
    title: '',
    summary: '',
    content: '',
    cover_image: '',
    category_id: undefined as number | undefined,
    status: 'draft' as 'draft' | 'published',
    scheduled_publish_at: '',
  });
  const primaryActionLabel = getArticlePrimaryActionLabel({
    isEdit,
    status: formData.status,
  });
  const scheduledPublishPickerValue = useMemo(
    () => toScheduledPickerValue(formData.scheduled_publish_at),
    [formData.scheduled_publish_at],
  );

  // 使用 Ref 实时跟踪最新数据，彻底解决 setInterval 闭包拿不到最新状态的问题
  const formDataRef = useRef(formData);
  useEffect(() => {
    formDataRef.current = formData;
  }, [formData]);

  const { data: categories, isError: isCategoriesError, error: categoriesError, refetch: refetchCategories } = useQuery({
    queryKey: ['categories'],
    queryFn: categoryApi.getCategories,
  });

  const { data: article, isLoading: isArticleLoading, isError: isArticleError, error: articleError, refetch: refetchArticle } = useQuery({
    queryKey: ['admin-article', id],
    queryFn: () => articleApi.getAdminArticleById(Number(id)),
    enabled: isEdit,
  });

  // 本地草稿 Key
  const draftKey = `wendao_draft_${id || 'new'}`;

  useEffect(() => {
    // 1. 先从本地恢复（针对新建文章或刷新）
    const localDraft = localStorage.getItem(draftKey);
    if (localDraft) {
      try {
        const parsed = JSON.parse(localDraft);
        setFormData((prev) => ({ ...prev, ...parsed }));
        setLastSavedTime(new Date().toLocaleTimeString() + ' (从本地恢复)');
      } catch (e) {
        console.error('Failed to parse local draft:', e);
      }
    }

    if (article) {
      setFormData({
        title: article.title,
        summary: article.summary,
        content: article.content,
        cover_image: article.cover_image || '',
        category_id: article.category_id,
        status: article.status,
        scheduled_publish_at: article.scheduled_publish_at || '',
      });
      lastSavedDataRef.current = { title: article.title, content: article.content, summary: article.summary };
    }
  }, [article, draftKey, id]); // 监听 id 变化

  // 实时备份到本地
  useEffect(() => {
    // 只有当有实质性内容时才备份到浏览器，防止初始化时存入空数据
    if (!formData.title && formData.content.length < 10) return;

    const backupData = {
      title: formData.title,
      summary: formData.summary,
      content: formData.content,
      category_id: formData.category_id,
      cover_image: formData.cover_image,
      scheduled_publish_at: formData.scheduled_publish_at,
    };
    localStorage.setItem(draftKey, JSON.stringify(backupData));
  }, [formData, draftKey]);

  useEffect(() => {
    // 只有草稿状态才开启自动保存
    if (formData.status !== 'draft') return;

    const timer = setInterval(async () => {
      // 必须从 Ref 中拿数据，闭包中的 formData 永远是旧的
      const { title, content, summary } = formDataRef.current;
      const isDirty = 
        title !== lastSavedDataRef.current.title || 
        content !== lastSavedDataRef.current.content ||
        summary !== lastSavedDataRef.current.summary;

      // 如果数据没变，或者内容太少（少于10个字且没标题），不触发后端保存
      if (!isDirty || isAutoSaving || (!title && content.length < 10)) return;

      setIsAutoSaving(true);
      try {
        if (isEdit) {
          // 已有文章：静默保存到数据库
          await articleApi.autoSave(Number(id), { title, content, summary });
          // 同步本地状态为草稿
          setFormData(prev => ({ ...prev, status: 'draft' }));
        } else {
          // 新建文章：持久化到数据库
          const newArticle = await articleApi.createArticle({
            ...formDataRef.current,
            title: title || '无标题草稿',
            status: 'draft'
          });
          // 成功后清除本地 'new' 缓存，并跳转到带 ID 的编辑页
          localStorage.removeItem(draftKey);
          navigate(`/admin/articles/edit/${newArticle.id}`, { replace: true });
        }
        lastSavedDataRef.current = { title, content, summary };
        setLastSavedTime(new Date().toLocaleTimeString());
      } catch (error) {
        console.error('Auto-save failed:', error);
      } finally {
        setIsAutoSaving(false);
      }
    }, 30000);

    return () => clearInterval(timer);
  }, [draftKey, formData.status, id, isAutoSaving, isEdit, navigate]);

  const saveMutation = useMutation({
    mutationFn: (data: typeof formData) =>
      isEdit ? articleApi.updateArticle(Number(id), data) : articleApi.createArticle(data),
    onSuccess: () => {
      localStorage.removeItem(draftKey);
      showToast(isEdit ? '文章已更新' : '文章已发布', 'success');
      navigate('/admin/articles');
    },
    onError: (error: any) => {
      showToast(error.message || '保存失败', 'error');
    },
  });

  const handleImageUpload = async (file: File, type: 'cover' | 'content') => {
    try {
      const res = await uploadApi.uploadImage(file, type);
      if (type === 'cover') {
        setFormData((prev) => ({ ...prev, cover_image: res.url }));
        showToast('封面上传成功', 'success');
      } else {
        const markdownImage = `\n![${res.filename}](${res.url})\n`;
        const textarea = contentInputRef.current;
        if (textarea) {
          const start = textarea.selectionStart;
          const end = textarea.selectionEnd;
          const newContent =
            formData.content.substring(0, start) +
            markdownImage +
            formData.content.substring(end);
          setFormData((prev) => ({ ...prev, content: newContent }));
          setTimeout(() => {
            textarea.focus();
            textarea.setSelectionRange(start + markdownImage.length, start + markdownImage.length);
          }, 0);
        } else {
          setFormData((prev) => ({ ...prev, content: prev.content + markdownImage }));
        }
        showToast('内容图片上传成功', 'success');
      }
    } catch (error: any) {
      showToast(error.message || '图片上传失败', 'error');
    }
  };

  const handleContentPaste = async (e: ClipboardEvent<HTMLTextAreaElement>) => {
    const items = e.clipboardData?.items;
    if (!items) return;

    for (const item of Array.from(items)) {
      if (item.kind === 'file' && item.type.startsWith('image/')) {
        const file = item.getAsFile();
        if (!file) return;

        e.preventDefault();
        showToast('检测到图片，正在上传...', 'info');
        await handleImageUpload(file, 'content');
        return;
      }
    }
  };

  const handleGenerateSummary = async () => {
    if (!formData.content.trim() || formData.content.length < 50) {
      showToast('文章内容太少，无法生成摘要（至少需要 50 个字符）', 'error');
      return;
    }

    setIsGeneratingSummary(true);
    try {
      const res = await chatApi.generateSummary(formData.content);
      setFormData((prev) => ({ ...prev, summary: res.summary }));
      showToast('摘要生成成功', 'success');
    } catch (error: any) {
      showToast(error.message || '生成摘要失败', 'error');
    } finally {
      setIsGeneratingSummary(false);
    }
  };

  const contentStats = useMemo(() => {
    const trimmed = formData.content.trim();
    const lineCount = formData.content ? formData.content.split('\n').length : 0;
    const cjkCount = (trimmed.match(/[\u4e00-\u9fff]/g) || []).length;
    const wordCount =
      trimmed
        .replace(/[\u4e00-\u9fff]/g, '')
        .split(/\s+/)
        .filter(Boolean).length + cjkCount;
    const readingMinutes = Math.max(1, Math.ceil(wordCount / 450));

    return {
      characters: formData.content.length,
      lines: lineCount,
      words: wordCount,
      readingMinutes,
    };
  }, [formData.content]);

  if (isEdit && isArticleLoading) return <Loading />;

  if (isEdit && isArticleError) {
    return (
      <div className="max-w-6xl mx-auto pb-12">
        <ErrorState
          message={(articleError as any)?.message || '文章加载失败'}
          onRetry={() => refetchArticle()}
        />
      </div>
    );
  }

  if (isCategoriesError) {
    return (
      <div className="max-w-6xl mx-auto pb-12">
        <ErrorState
          message={(categoriesError as any)?.message || '分类列表加载失败'}
          onRetry={() => refetchCategories()}
        />
      </div>
    );
  }

  return (
    <div
      className={`${isWritingFocused ? 'max-w-display' : 'max-w-6xl'} mx-auto pb-12 transition-[max-width] duration-300`}
    >
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-neutral-700 dark:text-neutral-100">
          {isEdit ? '编辑文章' : '新建文章'}
        </h1>
        <div className="space-x-4">
          <button
            type="button"
            onClick={() => navigate('/admin/articles')}
            className="btn btn-secondary"
          >
            取消
          </button>
          <button
            type="button"
            onClick={() => {
              if (!formData.title.trim()) {
                showToast('请输入标题', 'error');
                return;
              }
              if (!formData.category_id) {
                showToast('请选择文章分类', 'error');
                return;
              }
              if (formData.content.length < 10) {
                showToast('文章内容太少（至少10个字符）', 'error');
                return;
              }
              saveMutation.mutate(formData);
            }}
            disabled={saveMutation.isPending}
            className="btn btn-primary"
          >
            {saveMutation.isPending ? '保存中...' : primaryActionLabel}
          </button>
        </div>
      </div>

      <div
        className={
          isWritingFocused
            ? 'space-y-5'
            : 'space-y-6 rounded-xl border border-neutral-100 bg-white p-8 shadow-sm dark:border-neutral-800 dark:bg-neutral-900'
        }
      >
        <div
          className={
            isWritingFocused
              ? 'rounded-2xl border border-neutral-100 bg-white p-5 shadow-sm dark:border-neutral-800 dark:bg-neutral-900'
              : ''
          }
        >
          <div
            className={`grid grid-cols-1 gap-6 ${
              isWritingFocused ? 'xl:grid-cols-[minmax(0,1fr)_280px]' : 'lg:grid-cols-3'
            }`}
          >
          <div className={isWritingFocused ? 'space-y-4' : 'lg:col-span-2 space-y-6'}>
            <div>
              <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-1">标题</label>
              <input
                type="text"
                className="input w-full"
                value={formData.title}
                onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                placeholder="请输入文章标题"
              />
            </div>

            <div className="grid grid-cols-2 gap-6">
              <div>
                <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">分类</label>
                <Select
                  value={formData.category_id || undefined}
                  onChange={(value) => setFormData({ ...formData, category_id: value as number })}
                  placeholder="请选择分类"
                  style={{ width: '100%' }}
                >
                  {categories?.map((c) => (
                    <Select.Option key={c.id} value={c.id} label={c.name} />
                  ))}
                </Select>
              </div>
              <div>
                <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">状态</label>
                <Select
                  value={formData.status}
                  onChange={(value) => setFormData({ ...formData, status: value as 'draft' | 'published' })}
                  placeholder="请选择状态"
                  style={{ width: '100%' }}
                >
                  <Select.Option value="draft" label="草稿" />
                  <Select.Option value="published" label="发布" />
                </Select>
              </div>
            </div>

            <div>
              <div className="mb-2 flex items-center justify-between gap-3">
                <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300">定时发布</label>
                <span className="rounded-full bg-primary-50 px-2.5 py-1 text-[11px] font-semibold text-primary-600 ring-1 ring-primary-100 dark:bg-primary-500/10 dark:text-primary-300 dark:ring-primary-500/20">
                  {formData.scheduled_publish_at ? '已设置' : '可选'}
                </span>
              </div>
              <div className="article-schedule-picker rounded-2xl border border-neutral-100 bg-neutral-50/80 p-3 shadow-inner shadow-white/70 transition-colors dark:border-neutral-800 dark:bg-neutral-800/40 dark:shadow-none">
                <DatePicker
                  value={scheduledPublishPickerValue}
                  onChange={(value) =>
                    setFormData({
                      ...formData,
                      scheduled_publish_at: toScheduledPayloadValue(value),
                    })
                  }
                  enableTimePicker
                  valueType="YYYY-MM-DD HH:mm"
                  format={SCHEDULE_PICKER_FORMAT}
                  clearable
                  needConfirm
                  placeholder="选择自动发布时间"
                  prefixIcon={<CalendarClock className="h-4 w-4 text-primary-500 dark:text-primary-300" />}
                  size="large"
                  style={{ width: '100%' }}
                  className="w-full [&_.t-input]:rounded-xl [&_.t-input]:border-neutral-200 [&_.t-input]:bg-white [&_.t-input]:shadow-sm dark:[&_.t-input]:border-neutral-700 dark:[&_.t-input]:bg-neutral-900"
                />
              </div>
              <p className="mt-2 text-xs leading-5 text-neutral-400 dark:text-neutral-500">
                设置后文章将在指定时间自动发布。留空则立即发布。
              </p>
            </div>

            <div>
              <div className="flex justify-between items-center mb-1">
                <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300">摘要</label>
                <button
                  type="button"
                  onClick={handleGenerateSummary}
                  disabled={isGeneratingSummary}
                  className="text-xs text-primary-600 dark:text-primary-400 hover:text-primary-700 flex items-center gap-1 transition-colors disabled:opacity-50"
                >
                  {isGeneratingSummary ? (
                    <>
                      <svg className="animate-spin h-3 w-3" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                      </svg>
                      生成中...
                    </>
                  ) : (
                    <>
                      <svg xmlns="http://www.w3.org/2000/svg" className="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                      </svg>
                      AI 生成摘要
                    </>
                  )}
                </button>
              </div>
              <textarea
                className={`input w-full py-2 ${isWritingFocused ? 'h-20' : 'h-24'}`}
                value={formData.summary}
                onChange={(e) => setFormData({ ...formData, summary: e.target.value })}
                placeholder="请输入文章摘要，或点击上方按钮使用 AI 生成"
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">封面图</label>
            <div className="relative aspect-video rounded-lg overflow-hidden bg-neutral-100 dark:bg-neutral-800 border-2 border-dashed border-neutral-200 dark:border-neutral-700 hover:border-primary-300 transition-colors cursor-pointer group">
              {formData.cover_image ? (
                <>
                  <img
                    src={formData.cover_image}
                    alt="Cover"
                    className="w-full h-full object-cover"
                  />
                  <div className="absolute inset-0 bg-black/40 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center gap-2">
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        setFormData({ ...formData, cover_image: '' });
                      }}
                      className="p-2 bg-red-500 text-white rounded-full hover:bg-red-600"
                    >
                      <svg xmlns="http://www.w3.org/2000/svg" className="h-5 w-5" viewBox="0 0 20 20" fill="currentColor">
                        <path fillRule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clipRule="evenodd" />
                      </svg>
                    </button>
                  </div>
                </>
              ) : (
                <div className="absolute inset-0 flex flex-col items-center justify-center text-neutral-400 dark:text-neutral-500">
                  <svg xmlns="http://www.w3.org/2000/svg" className="h-10 w-10 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
                  </svg>
                  <span className="text-sm">点击上传封面</span>
                </div>
              )}
              <input
                type="file"
                className="absolute inset-0 opacity-0 cursor-pointer"
                accept="image/*"
                onChange={(e) => {
                  const file = e.target.files?.[0];
                  if (file) handleImageUpload(file, 'cover');
                }}
              />
            </div>
            <p className="mt-2 text-xs text-neutral-400 dark:text-neutral-500">支持 jpg、png、webp 等格式，建议比例 16:9</p>
          </div>
        </div>

        </div>

        <MarkdownWritingStudio
          content={formData.content}
          onContentChange={(content) => setFormData((prev) => ({ ...prev, content }))}
          textareaRef={contentInputRef}
          onPaste={handleContentPaste}
          onImageUploadClick={() => contentImageInputRef.current?.click()}
          contentStats={contentStats}
          lastSavedTime={lastSavedTime}
          isAutoSaving={isAutoSaving}
          isImmersive={isWritingFocused}
          onImmersiveChange={setIsWritingFocused}
        />

        <input
          ref={contentImageInputRef}
          type="file"
          className="hidden"
          accept="image/*"
          onChange={(event: ChangeEvent<HTMLInputElement>) => {
            const file = event.target.files?.[0];
            if (file) handleImageUpload(file, 'content');
            event.target.value = '';
          }}
        />
      </div>
    </div>
  );
};
