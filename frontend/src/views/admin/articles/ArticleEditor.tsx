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
import { useTranslation } from 'react-i18next';
import { articleApi, categoryApi, collectionApi, tagApi, uploadApi, chatApi } from '@/api';
import type { AIWritingAction } from '@/api/chat';
import { Loading, ErrorState } from '@/components/common';
import { useUIStore } from '@/store';
import { getArticlePrimaryActionLabel } from '@/utils/pageBehavior';
import { MarkdownWritingStudio } from './components/MarkdownWritingStudio';
import 'tdesign-react/es/style/index.css';

const SCHEDULE_PICKER_FORMAT = 'YYYY-MM-DD HH:mm';
type ScheduledPickerValue = Parameters<NonNullable<DatePickerProps['onChange']>>[0];
type AIWritingPanelState = {
  action: AIWritingAction;
  isGenerating: boolean;
  result: string;
  suggestions: string[];
  selectionStart: number;
  selectionEnd: number;
  selectedText: string;
} | null;
type AISummaryPanelState = {
  isGenerating: boolean;
  result: string;
} | null;

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
  const { t } = useTranslation();
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { showToast } = useUIStore();
  const isEdit = !!id;
  const contentInputRef = useRef<HTMLTextAreaElement>(null);
  const contentImageInputRef = useRef<HTMLInputElement>(null);
  const [lastSavedTime, setLastSavedTime] = useState<string | null>(null);
  const [isAutoSaving, setIsAutoSaving] = useState(false);
  const [isWritingFocused, setIsWritingFocused] = useState(false);
  const [aiSummaryPanel, setAISummaryPanel] = useState<AISummaryPanelState>(null);
  const [aiWritingPanel, setAIWritingPanel] = useState<AIWritingPanelState>(null);
  const lastSavedDataRef = useRef({ title: '', content: '', summary: '' });

  const [formData, setFormData] = useState({
    title: '',
    summary: '',
    content: '',
    cover_image: '',
    category_id: undefined as number | undefined,
    status: 'draft' as 'draft' | 'published',
    scheduled_publish_at: '',
    collection_id: undefined as number | undefined,
    collection_position: 0,
    tag_ids: [] as number[],
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

  const {
    data: collections,
    isError: isCollectionsError,
    error: collectionsError,
    refetch: refetchCollections,
  } = useQuery({
    queryKey: ['collections'],
    queryFn: collectionApi.getCollections,
  });

  const { data: tags, isError: isTagsError, error: tagsError, refetch: refetchTags } = useQuery({
    queryKey: ['tags'],
    queryFn: tagApi.getTags,
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
        setLastSavedTime(new Date().toLocaleTimeString() + ` ${t('articleEditor.draftRecovered')}`);
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
        collection_id: article.collection_membership?.collection_id,
        collection_position: article.collection_membership?.position ?? 0,
        tag_ids: article.tags?.map((tag) => tag.id) ?? [],
      });
      lastSavedDataRef.current = { title: article.title, content: article.content, summary: article.summary };
    }
  }, [article, draftKey, id, t]); // 监听 id 变化

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
      collection_id: formData.collection_id,
      collection_position: formData.collection_position,
      tag_ids: formData.tag_ids,
    };
    localStorage.setItem(draftKey, JSON.stringify(backupData));
  }, [formData, draftKey, t]);

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
            title: title || t('articleEditor.noTitleDraft'),
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
  }, [draftKey, formData.status, id, isAutoSaving, isEdit, navigate, t]);

  const saveMutation = useMutation({
    mutationFn: (data: typeof formData) =>
      isEdit ? articleApi.updateArticle(Number(id), data) : articleApi.createArticle(data),
    onSuccess: () => {
      localStorage.removeItem(draftKey);
      showToast(isEdit ? t('articleEditor.articleUpdated') : t('articleEditor.articlePublished'), 'success');
      navigate('/admin/articles');
    },
    onError: (error: any) => {
      showToast(error.message || t('articleEditor.saveFailed'), 'error');
    },
  });

  const handleImageUpload = async (file: File, type: 'cover' | 'content') => {
    try {
      const res = await uploadApi.uploadImage(file, type);
      if (type === 'cover') {
        setFormData((prev) => ({ ...prev, cover_image: res.url }));
        showToast(t('articleEditor.coverUploadSuccess'), 'success');
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
        showToast(t('articleEditor.contentImageUploadSuccess'), 'success');
      }
    } catch (error: any) {
      showToast(error.message || t('articleEditor.imageUploadFailed'), 'error');
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
        showToast(t('articleEditor.imageUploading'), 'info');
        await handleImageUpload(file, 'content');
        return;
      }
    }
  };

  const captureAISelection = () => {
    const textarea = contentInputRef.current;
    if (!textarea) {
      return { start: 0, end: 0, text: '' };
    }

    const start = textarea.selectionStart ?? 0;
    const end = textarea.selectionEnd ?? start;
    return {
      start,
      end,
      text: formData.content.slice(start, end),
    };
  };

  const handleGenerateSummary = async () => {
    if (!formData.content.trim() || formData.content.length < 50) {
      showToast(t('articleEditor.contentTooShort'), 'error');
      return;
    }

    setAIWritingPanel(null);
    setAISummaryPanel({
      isGenerating: true,
      result: '',
    });
    try {
      const res = await chatApi.generateSummary(formData.content);
      setAISummaryPanel({
        isGenerating: false,
        result: res.summary,
      });
      showToast(t('articleEditor.summarySuccess'), 'success');
    } catch (error: any) {
      setAISummaryPanel(null);
      showToast(error.message || t('articleEditor.summaryFailed'), 'error');
    }
  };

  const handleAISummaryApply = () => {
    if (!aiSummaryPanel) return;
    setFormData((prev) => ({ ...prev, summary: aiSummaryPanel.result }));
    setAISummaryPanel(null);
    showToast(t('articleEditor.summaryApplied'), 'success');
  };

  const handleAIWritingAction = async (action: AIWritingAction) => {
    const currentSelection = captureAISelection();
    setAISummaryPanel(null);
    const { start: selectionStart, end: selectionEnd, text: selectedText } = currentSelection;
    const isTitleAction = action === 'seo-title';
    const content = isTitleAction ? formData.content : selectedText;

    if (!isTitleAction && !selectedText.trim()) {
      showToast(t('articleEditor.aiWritingSelectTextFirst'), 'error');
      return;
    }
    if (!content.trim() || content.trim().length < 10) {
      showToast(t('articleEditor.aiWritingContentTooShort'), 'error');
      return;
    }

    setAIWritingPanel({
      action,
      isGenerating: true,
      result: '',
      suggestions: [],
      selectionStart,
      selectionEnd,
      selectedText,
    });

    try {
      const res = await chatApi.generateWriting({
        action,
        content,
        title: formData.title,
        summary: formData.summary,
      });
      setAIWritingPanel({
        action,
        isGenerating: false,
        result: res.result,
        suggestions: res.suggestions ?? [],
        selectionStart,
        selectionEnd,
        selectedText,
      });
      showToast(t('articleEditor.aiWritingSuccess'), 'success');
    } catch (error: any) {
      setAIWritingPanel(null);
      showToast(error.message || t('articleEditor.aiWritingFailed'), 'error');
    }
  };

  const handleAIWritingApply = (appliedText: string) => {
    if (!aiWritingPanel) return;

    if (aiWritingPanel.action === 'seo-title') {
      setFormData((prev) => ({ ...prev, title: appliedText }));
      setAIWritingPanel(null);
      showToast(t('articleEditor.aiWritingApplied'), 'success');
      return;
    }

    const { selectionStart, selectionEnd } = aiWritingPanel;
    if (formData.content.slice(selectionStart, selectionEnd) !== aiWritingPanel.selectedText) {
      showToast(t('articleEditor.aiWritingSelectionChanged'), 'error');
      setAIWritingPanel(null);
      return;
    }

    const nextContent =
      formData.content.slice(0, selectionStart) +
      appliedText +
      formData.content.slice(selectionEnd);
    setFormData((prev) => ({ ...prev, content: nextContent }));
    setAIWritingPanel(null);

    requestAnimationFrame(() => {
      const textarea = contentInputRef.current;
      if (!textarea) return;
      const cursor = selectionStart + appliedText.length;
      textarea.focus();
      textarea.setSelectionRange(cursor, cursor);
    });
    showToast(t('articleEditor.aiWritingApplied'), 'success');
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
          message={(articleError as any)?.message || t('articleEditor.articleLoadFailed')}
          onRetry={() => refetchArticle()}
        />
      </div>
    );
  }

  if (isCategoriesError) {
    return (
      <div className="max-w-6xl mx-auto pb-12">
        <ErrorState
          message={(categoriesError as any)?.message || t('articleEditor.categoryLoadFailed')}
          onRetry={() => refetchCategories()}
        />
      </div>
    );
  }

  if (isCollectionsError) {
    return (
      <div className="max-w-6xl mx-auto pb-12">
        <ErrorState
          message={(collectionsError as any)?.message || '合集列表加载失败'}
          onRetry={() => refetchCollections()}
        />
      </div>
    );
  }

  if (isTagsError) {
    return (
      <div className="max-w-6xl mx-auto pb-12">
        <ErrorState
          message={(tagsError as any)?.message || '标签列表加载失败'}
          onRetry={() => refetchTags()}
        />
      </div>
    );
  }

  return (
    <div
      className={`${isWritingFocused ? 'max-w-display' : 'max-w-6xl'} mx-auto pb-12 transition-[max-width] duration-300`}
    >
      <div className="mb-6 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <h1 className="text-2xl font-bold text-neutral-700 dark:text-neutral-100">
          {isEdit ? t('articleEditor.editArticle') : t('articleEditor.newArticle')}
        </h1>
        <div className="grid grid-cols-2 gap-2 sm:flex sm:items-center sm:gap-4">
          <button
            type="button"
            onClick={() => navigate('/admin/articles')}
            className="btn btn-secondary w-full sm:w-auto"
          >
            {t('articleEditor.cancel')}
          </button>
          <button
            type="button"
            onClick={() => {
              if (!formData.title.trim()) {
                showToast(t('articleEditor.titleRequired'), 'error');
                return;
              }
              if (!formData.category_id) {
                showToast(t('articleEditor.categoryRequired'), 'error');
                return;
              }
              if (formData.content.length < 10) {
                showToast(t('articleEditor.contentTooShortToSave'), 'error');
                return;
              }
              saveMutation.mutate(formData);
            }}
            disabled={saveMutation.isPending}
            className="btn btn-primary w-full sm:w-auto"
          >
            {saveMutation.isPending ? t('common.saving') : primaryActionLabel}
          </button>
        </div>
      </div>

      <div
        className={
          isWritingFocused
            ? 'space-y-5'
            : 'space-y-6 rounded-xl border border-neutral-100 bg-white p-4 shadow-sm dark:border-neutral-800 dark:bg-neutral-900 sm:p-8'
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
              <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-1">{t('articleEditor.titleLabel')}</label>
              <input
                type="text"
                className="input w-full"
                value={formData.title}
                onChange={(e) => setFormData({ ...formData, title: e.target.value })}
                placeholder={t('articleEditor.titlePlaceholder')}
              />
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 sm:gap-6">
              <div>
                <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">{t('articleEditor.categoryLabel')}</label>
                <Select
                  value={formData.category_id || undefined}
                  onChange={(value) => setFormData({ ...formData, category_id: value as number })}
                  placeholder={t('articleEditor.categoryPlaceholder')}
                  style={{ width: '100%' }}
                >
                  {categories?.map((c) => (
                    <Select.Option key={c.id} value={c.id} label={c.name} />
                  ))}
                </Select>
              </div>
              <div>
                <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">{t('articleEditor.statusLabel')}</label>
                <Select
                  value={formData.status}
                  onChange={(value) => setFormData({ ...formData, status: value as 'draft' | 'published' })}
                  placeholder={t('articleEditor.statusLabel')}
                  style={{ width: '100%' }}
                >
                  <Select.Option value="draft" label={t('articleEditor.statusDraft')} />
                  <Select.Option value="published" label={t('articleEditor.statusPublished')} />
                </Select>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-6 lg:grid-cols-[minmax(0,1fr)_160px]">
              <div>
                <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">所属合集</label>
                <Select
                  value={formData.collection_id || undefined}
                  onChange={(value) =>
                    setFormData({
                      ...formData,
                      collection_id: value ? (value as number) : undefined,
                    })
                  }
                  clearable
                  placeholder="不加入合集"
                  style={{ width: '100%' }}
                >
                  {collections?.map((collection) => (
                    <Select.Option key={collection.id} value={collection.id} label={collection.name} />
                  ))}
                </Select>
              </div>
              <div>
                <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">合集排序</label>
                <input
                  type="number"
                  min={0}
                  step={1}
                  className="input w-full"
                  value={formData.collection_position}
                  onChange={(e) =>
                    setFormData({
                      ...formData,
                      collection_position: Number(e.target.value),
                    })
                  }
                  disabled={!formData.collection_id}
                />
              </div>
            </div>

            <div>
              <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">文章标签</label>
              <Select
                value={formData.tag_ids}
                onChange={(value) =>
                  setFormData({
                    ...formData,
                    tag_ids: Array.isArray(value) ? (value as number[]) : [],
                  })
                }
                multiple
                clearable
                placeholder="选择标签"
                style={{ width: '100%' }}
              >
                {tags?.map((tag) => (
                  <Select.Option key={tag.id} value={tag.id} label={tag.name} />
                ))}
              </Select>
            </div>

            <div>
              <div className="mb-2 flex items-center justify-between gap-3">
                <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300">{t('articleEditor.scheduledPublish')}</label>
                <span className="rounded-full bg-primary-50 px-2.5 py-1 text-[11px] font-semibold text-primary-600 ring-1 ring-primary-100 dark:bg-primary-500/10 dark:text-primary-300 dark:ring-primary-500/20">
                  {formData.scheduled_publish_at ? t('articleEditor.scheduledSet') : t('articleEditor.optional')}
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
                  placeholder={t('articleEditor.scheduledPlaceholder')}
                  prefixIcon={<CalendarClock className="h-4 w-4 text-primary-500 dark:text-primary-300" />}
                  size="large"
                  style={{ width: '100%' }}
                  className="w-full [&_.t-input]:rounded-xl [&_.t-input]:border-neutral-200 [&_.t-input]:bg-white [&_.t-input]:shadow-sm dark:[&_.t-input]:border-neutral-700 dark:[&_.t-input]:bg-neutral-900"
                />
              </div>
              <p className="mt-2 text-xs leading-5 text-neutral-400 dark:text-neutral-500">
                {t('articleEditor.scheduledHint')}
              </p>
            </div>

            <div className="rounded-2xl border border-neutral-100 bg-white p-4 shadow-sm transition-colors dark:border-neutral-800 dark:bg-neutral-900">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300">
                    {t('articleEditor.summaryLabel')}
                  </label>
                  <p className="mt-1 text-xs text-neutral-400 dark:text-neutral-500">
                    {t('articleEditor.summaryPlaceholder')}
                  </p>
                </div>
              </div>
              <textarea
                className={`input mt-3 w-full py-2 ${isWritingFocused ? 'h-20' : 'h-24'}`}
                value={formData.summary}
                onChange={(e) => setFormData({ ...formData, summary: e.target.value })}
                placeholder={t('articleEditor.summaryPlaceholder')}
              />
            </div>
          </div>

          <div>
            <label className="block text-sm font-medium text-neutral-700 dark:text-neutral-300 mb-2">{t('articleEditor.coverLabel')}</label>
            <div className="relative aspect-video rounded-lg overflow-hidden bg-neutral-100 dark:bg-neutral-800 border-2 border-dashed border-neutral-200 dark:border-neutral-700 hover:border-primary-300 transition-colors cursor-pointer group">
              {formData.cover_image ? (
                <>
                  <img
                    src={formData.cover_image}
                    alt={t('articleEditor.coverLabel')}
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
                  <span className="text-sm">{t('articleEditor.clickUploadCover')}</span>
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
            <p className="mt-2 text-xs text-neutral-400 dark:text-neutral-500">{t('articleEditor.coverHint')}</p>
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
          aiSummaryPanel={aiSummaryPanel}
          aiWritingPanel={aiWritingPanel}
          onGenerateSummary={handleGenerateSummary}
          onApplySummary={handleAISummaryApply}
          onGenerateWritingAction={handleAIWritingAction}
          onApplyWritingResult={handleAIWritingApply}
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
