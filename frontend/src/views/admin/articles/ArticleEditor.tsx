import { type ChangeEvent, Suspense, lazy, useEffect, useMemo, useRef, useState } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useQuery, useMutation } from '@tanstack/react-query';
import { Select } from 'tdesign-react';
import {
  Bold,
  Code,
  Eye,
  Heading2,
  ImagePlus,
  Link as LinkIcon,
  List,
  ListOrdered,
  Minus,
  PanelLeft,
  Pilcrow,
  Quote,
  SplitSquareHorizontal,
  type LucideIcon,
} from 'lucide-react';
import { articleApi, categoryApi, uploadApi, chatApi } from '@/api';
import { Loading } from '@/components/common';
import { useUIStore } from '@/store';
import { getArticlePrimaryActionLabel } from '@/utils/pageBehavior';
import {
  applyMarkdownAction,
  type ApplyMarkdownActionResult,
  type MarkdownAction,
} from '@/utils/markdownEditor';
import 'tdesign-react/es/style/index.css';

const ArticlePreview = lazy(() =>
  import('./ArticlePreview').then((module) => ({ default: module.ArticlePreview }))
);

type EditorMode = 'edit' | 'split' | 'preview';

const markdownToolbarActions: Array<{
  action: MarkdownAction;
  label: string;
  icon: LucideIcon;
}> = [
  { action: 'heading', label: '二级标题', icon: Heading2 },
  { action: 'bold', label: '加粗', icon: Bold },
  { action: 'quote', label: '引用', icon: Quote },
  { action: 'unordered-list', label: '无序列表', icon: List },
  { action: 'ordered-list', label: '有序列表', icon: ListOrdered },
  { action: 'inline-code', label: '行内代码', icon: Code },
  { action: 'code-block', label: '代码块', icon: Pilcrow },
  { action: 'link', label: '链接', icon: LinkIcon },
  { action: 'divider', label: '分割线', icon: Minus },
];

const restoreTextareaSelection = (
  textarea: HTMLTextAreaElement,
  result: ApplyMarkdownActionResult
) => {
  requestAnimationFrame(() => {
    textarea.focus();
    textarea.setSelectionRange(result.selection.start, result.selection.end);
  });
};

const insertMarkdownWithUndoStack = (
  textarea: HTMLTextAreaElement,
  result: ApplyMarkdownActionResult
) => {
  textarea.focus();
  textarea.setSelectionRange(result.edit.start, result.edit.end);

  try {
    const canInsertText =
      !document.queryCommandSupported || document.queryCommandSupported('insertText');
    if (!canInsertText) return false;

    const didInsert = document.execCommand('insertText', false, result.edit.replacement);
    if (!didInsert) return false;

    const inputEvent =
      typeof InputEvent === 'function'
        ? new InputEvent('input', {
            bubbles: true,
            inputType: 'insertText',
            data: result.edit.replacement,
          })
        : new Event('input', { bubbles: true });
    textarea.dispatchEvent(inputEvent);
    restoreTextareaSelection(textarea, result);
    return true;
  } catch {
    return false;
  }
};

export const ArticleEditor = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { showToast } = useUIStore();
  const isEdit = !!id;
  const contentInputRef = useRef<HTMLTextAreaElement>(null);
  const contentImageInputRef = useRef<HTMLInputElement>(null);
  const [editorMode, setEditorMode] = useState<EditorMode>('split');
  const [isGeneratingSummary, setIsGeneratingSummary] = useState(false);
  const [lastSavedTime, setLastSavedTime] = useState<string | null>(null);
  const [isAutoSaving, setIsAutoSaving] = useState(false);
  const lastSavedDataRef = useRef({ title: '', content: '', summary: '' });

  const [formData, setFormData] = useState({
    title: '',
    summary: '',
    content: '',
    cover_image: '',
    category_id: undefined as number | undefined,
    status: 'draft' as 'draft' | 'published',
  });
  const primaryActionLabel = getArticlePrimaryActionLabel({
    isEdit,
    status: formData.status,
  });

  // 使用 Ref 实时跟踪最新数据，彻底解决 setInterval 闭包拿不到最新状态的问题
  const formDataRef = useRef(formData);
  useEffect(() => {
    formDataRef.current = formData;
  }, [formData]);

  const { data: categories } = useQuery({
    queryKey: ['categories'],
    queryFn: categoryApi.getCategories,
  });

  const { data: article, isLoading: isArticleLoading } = useQuery({
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

  const handleMarkdownAction = (action: MarkdownAction) => {
    const textarea = contentInputRef.current;
    const selectionStart = textarea?.selectionStart ?? formData.content.length;
    const selectionEnd = textarea?.selectionEnd ?? selectionStart;

    const result = applyMarkdownAction({
      text: formData.content,
      selectionStart,
      selectionEnd,
      action,
    });

    if (textarea && insertMarkdownWithUndoStack(textarea, result)) return;

    setFormData((prev) => ({ ...prev, content: result.text }));
    if (textarea) restoreTextareaSelection(textarea, result);
  };

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

  const handleContentPaste = async (e: React.ClipboardEvent<HTMLTextAreaElement>) => {
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

  return (
    <div className="max-w-6xl mx-auto pb-12">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-neutral-700 dark:text-neutral-100">
          {isEdit ? '编辑文章' : '新建文章'}
        </h1>
        <div className="space-x-4">
          <button
            onClick={() => navigate('/admin/articles')}
            className="btn btn-secondary"
          >
            取消
          </button>
          <button
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

      <div className="space-y-6 bg-white dark:bg-neutral-900 p-8 rounded-xl shadow-sm border border-neutral-100 dark:border-neutral-800">
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-2 space-y-6">
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
                className="input w-full h-24 py-2"
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

        <div className="space-y-3">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <label className="block text-sm font-semibold text-neutral-700 dark:text-neutral-200">内容 (Markdown)</label>
              <p className="mt-1 text-xs text-neutral-400 dark:text-neutral-500">
                支持工具栏插入常用 Markdown，粘贴图片会自动上传。
              </p>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              {(['edit', 'split', 'preview'] as const).map((mode) => {
                const Icon =
                  mode === 'edit' ? PanelLeft : mode === 'split' ? SplitSquareHorizontal : Eye;
                return (
                  <button
                    key={mode}
                    type="button"
                    onClick={() => setEditorMode(mode)}
                    className={`inline-flex h-9 items-center gap-2 rounded-xl px-3 text-xs font-semibold transition-colors ${
                      editorMode === mode
                        ? 'bg-neutral-900 text-white dark:bg-neutral-100 dark:text-neutral-900'
                        : 'bg-neutral-100 text-neutral-500 hover:text-neutral-900 dark:bg-neutral-800 dark:text-neutral-400 dark:hover:text-neutral-100'
                    }`}
                  >
                    <Icon className="h-4 w-4" />
                    {mode === 'edit' ? '编辑' : mode === 'split' ? '分屏' : '预览'}
                  </button>
                );
              })}
            </div>
          </div>

          <div className="rounded-2xl border border-neutral-200 bg-neutral-50/70 p-3 dark:border-neutral-800 dark:bg-neutral-950/40">
            <div className="mb-3 flex flex-wrap items-center gap-2">
              {markdownToolbarActions.map((item) => (
                <button
                  key={item.action}
                  type="button"
                  aria-label={item.label}
                  onClick={() => handleMarkdownAction(item.action)}
                  className="group relative inline-flex h-9 w-9 items-center justify-center rounded-xl text-neutral-500 transition-colors hover:bg-white hover:text-neutral-900 focus-visible:bg-white focus-visible:text-neutral-900 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:text-neutral-400 dark:hover:bg-neutral-800 dark:hover:text-neutral-100 dark:focus-visible:bg-neutral-800 dark:focus-visible:text-neutral-100"
                >
                  <item.icon className="h-4 w-4" aria-hidden="true" />
                  <span className="pointer-events-none absolute left-1/2 top-full z-30 mt-2 -translate-x-1/2 whitespace-nowrap rounded-md border border-neutral-200 bg-white px-2 py-1 text-[11px] font-medium text-neutral-600 opacity-0 shadow-sm transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200">
                    {item.label}
                  </span>
                </button>
              ))}
              <button
                type="button"
                aria-label="插入图片"
                onClick={() => contentImageInputRef.current?.click()}
                className="group relative inline-flex h-9 items-center gap-2 rounded-xl px-3 text-xs font-semibold text-primary-600 transition-colors hover:bg-white focus-visible:bg-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-500/30 dark:text-primary-400 dark:hover:bg-neutral-800 dark:focus-visible:bg-neutral-800"
              >
                <ImagePlus className="h-4 w-4" aria-hidden="true" />
                图片
                <span className="pointer-events-none absolute left-1/2 top-full z-30 mt-2 -translate-x-1/2 whitespace-nowrap rounded-md border border-neutral-200 bg-white px-2 py-1 text-[11px] font-medium text-neutral-600 opacity-0 shadow-sm transition-opacity group-hover:opacity-100 group-focus-visible:opacity-100 dark:border-neutral-700 dark:bg-neutral-900 dark:text-neutral-200">
                  插入图片
                </span>
              </button>
              <input
                ref={contentImageInputRef}
                type="file"
                className="hidden"
                accept="image/*"
                onChange={(e: ChangeEvent<HTMLInputElement>) => {
                  const file = e.target.files?.[0];
                  if (file) handleImageUpload(file, 'content');
                  e.target.value = '';
                }}
              />
              <div className="ml-auto flex flex-wrap items-center gap-3 text-[11px] font-medium text-neutral-400 dark:text-neutral-500">
                <span>{contentStats.characters} 字符</span>
                <span>{contentStats.lines} 行</span>
                <span>{contentStats.words} 词</span>
                <span>约 {contentStats.readingMinutes} 分钟</span>
              </div>
            </div>

            <div className={`grid gap-4 ${editorMode === 'split' ? 'lg:grid-cols-2' : 'grid-cols-1'}`}>
              {editorMode !== 'preview' && (
                <section className="flex min-h-[540px] flex-col overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
                  <div className="flex items-center justify-between border-b border-neutral-100 px-4 py-3 dark:border-neutral-800">
                    <div>
                      <div className="text-sm font-semibold text-neutral-700 dark:text-neutral-200">Markdown</div>
                      <div className="text-[11px] text-neutral-400 dark:text-neutral-500">原文编辑</div>
                    </div>
                    {lastSavedTime && (
                      <div className="flex items-center gap-2 text-[10px] font-bold uppercase tracking-wider text-neutral-400">
                        <div className={`h-1.5 w-1.5 rounded-full ${isAutoSaving ? 'animate-pulse bg-amber-400' : 'bg-emerald-400'}`} />
                        {isAutoSaving ? '正在自动保存' : `已保存 ${lastSavedTime}`}
                      </div>
                    )}
                  </div>
                  <textarea
                    ref={contentInputRef}
                    className="admin-markdown-editor min-h-[480px] flex-1 resize-none border-0 bg-transparent px-5 py-4 text-sm leading-7 text-neutral-800 outline-none dark:text-neutral-100"
                    value={formData.content}
                    onChange={(e) => setFormData({ ...formData, content: e.target.value })}
                    onPaste={handleContentPaste}
                    placeholder="使用 Markdown 编写内容..."
                  />
                </section>
              )}

              {editorMode !== 'edit' && (
                <section className="flex min-h-[540px] flex-col overflow-hidden rounded-2xl border border-neutral-200 bg-white shadow-sm dark:border-neutral-800 dark:bg-neutral-900">
                  <div className="flex items-center justify-between border-b border-neutral-100 px-4 py-3 dark:border-neutral-800">
                    <div>
                      <div className="text-sm font-semibold text-neutral-700 dark:text-neutral-200">Preview</div>
                      <div className="text-[11px] text-neutral-400 dark:text-neutral-500">编辑时预览</div>
                    </div>
                  </div>
                  <div className="article-reading-body admin-markdown-preview flex-1 overflow-y-auto px-6 py-5">
                    {formData.content.trim() ? (
                      <Suspense fallback={<div className="h-full animate-pulse rounded-xl bg-neutral-100 dark:bg-neutral-800" />}>
                        <ArticlePreview content={formData.content} />
                      </Suspense>
                    ) : (
                      <div className="flex h-full min-h-[420px] items-center justify-center rounded-xl border border-dashed border-neutral-200 text-sm text-neutral-400 dark:border-neutral-800 dark:text-neutral-500">
                        预览会在这里显示
                      </div>
                    )}
                  </div>
                </section>
              )}
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
