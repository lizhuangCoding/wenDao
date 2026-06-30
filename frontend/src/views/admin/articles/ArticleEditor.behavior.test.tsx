import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { ArticleEditor } from './ArticleEditor';

const navigateMock = vi.hoisted(() => vi.fn());
const showToastMock = vi.hoisted(() => vi.fn());
const apiMocks = vi.hoisted(() => ({
  articleApi: {
    autoSave: vi.fn(),
    createArticle: vi.fn(),
    getAdminArticleById: vi.fn(),
    updateArticle: vi.fn(),
  },
  categoryApi: {
    getCategories: vi.fn(),
  },
  collectionApi: {
    getCollections: vi.fn(),
  },
  tagApi: {
    getTags: vi.fn(),
  },
  uploadApi: {
    uploadImage: vi.fn(),
  },
  chatApi: {
    generateSummary: vi.fn(),
    generateWriting: vi.fn(),
  },
}));

const paramsState = vi.hoisted<{ id?: string }>(() => ({}));

vi.mock('react-i18next', () => ({
  initReactI18next: {
    type: '3rdParty',
    init: () => {},
  },
  useTranslation: () => ({
    t: (key: string, options?: { time?: string }) => (options?.time ? `${key}:${options.time}` : key),
  }),
}));

vi.mock('react-router-dom', () => ({
  useNavigate: () => navigateMock,
  useParams: () => paramsState,
}));

vi.mock('tdesign-react', () => {
  const Select = ({ value, onChange, children, multiple = false }: any) => (
    <select
      data-testid="mock-select"
      multiple={multiple}
      value={multiple ? (Array.isArray(value) ? value.map(String) : []) : (value ?? '')}
      onChange={(event) => {
        if (multiple) {
          const nextValues = Array.from(event.target.selectedOptions).map((option) => {
            const numericValue = Number(option.value);
            return Number.isNaN(numericValue) ? option.value : numericValue;
          });
          onChange?.(nextValues);
          return;
        }

        const nextValue = event.target.value;
        if (nextValue === '') {
          onChange?.(undefined);
          return;
        }
        const numericValue = Number(nextValue);
        onChange?.(Number.isNaN(numericValue) ? nextValue : numericValue);
      }}
    >
      {children}
    </select>
  );

  Select.Option = ({ value, label }: any) => <option value={value}>{label}</option>;

  const DatePicker = ({ value, onChange, placeholder }: any) => (
    <input
      data-testid="mock-date-picker"
      value={value ?? ''}
      placeholder={placeholder}
      onChange={(event) => onChange?.(event.target.value)}
    />
  );

  return {
    Select,
    DatePicker,
  };
});

vi.mock('@/api', () => apiMocks);

vi.mock('@/components/common', () => ({
  Loading: () => <div>loading</div>,
  ErrorState: ({ message }: { message: string }) => <div>{message}</div>,
}));

vi.mock('@/store', () => ({
  useUIStore: () => ({
    showToast: showToastMock,
  }),
}));

vi.mock('./components/MarkdownWritingStudio', () => ({
  MarkdownWritingStudio: ({
    content,
    onContentChange,
    onImageUploadClick,
    textareaRef,
  }: {
    content: string;
    onContentChange: (content: string) => void;
    onImageUploadClick: () => void;
    textareaRef: React.RefObject<HTMLTextAreaElement>;
  }) => (
    <div>
      <button type="button" onClick={onImageUploadClick}>
        trigger-content-upload
      </button>
      <textarea
        aria-label="content-editor"
        ref={textareaRef}
        value={content}
        onChange={(event) => onContentChange(event.target.value)}
      />
    </div>
  ),
}));

const renderEditor = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
      },
      mutations: {
        retry: false,
      },
    },
  });

  return render(
    <QueryClientProvider client={queryClient}>
      <ArticleEditor />
    </QueryClientProvider>
  );
};

describe('ArticleEditor behavior', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.useRealTimers();
    paramsState.id = undefined;
    localStorage.clear();
    apiMocks.categoryApi.getCategories.mockResolvedValue([{ id: 1, name: '分类' }]);
    apiMocks.collectionApi.getCollections.mockResolvedValue([]);
    apiMocks.tagApi.getTags.mockResolvedValue([]);
    apiMocks.articleApi.getAdminArticleById.mockResolvedValue({
      id: 42,
      title: '旧标题',
      summary: '旧摘要',
      content: '已有内容已经超过十个字',
      cover_image: '',
      category_id: 1,
      status: 'draft',
      scheduled_publish_at: '',
      collection_membership: null,
      tags: [],
    });
    apiMocks.articleApi.autoSave.mockResolvedValue({});
    apiMocks.articleApi.createArticle.mockResolvedValue({
      id: 99,
      title: '新草稿',
    });
  });

  it('uploads a content image and inserts markdown into the editor', async () => {
    apiMocks.uploadApi.uploadImage.mockResolvedValue({
      url: '/uploads/test-image.png',
      filename: 'test-image.png',
      size: 12,
    });

    const { container } = renderEditor();

    const textarea = await screen.findByLabelText('content-editor');
    fireEvent.change(textarea, { target: { value: '正文内容' } });

    const contentUploadInput = container.querySelector('input.hidden[type="file"]');
    if (!(contentUploadInput instanceof HTMLInputElement)) {
      throw new Error('content upload input not found');
    }

    const file = new File(['image'], 'test-image.png', { type: 'image/png' });
    fireEvent.change(contentUploadInput, { target: { files: [file] } });

    await waitFor(() => {
      expect(String((screen.getByLabelText('content-editor') as HTMLTextAreaElement).value)).toContain(
        '![test-image.png](/uploads/test-image.png)'
      );
    });

    expect(apiMocks.uploadApi.uploadImage).toHaveBeenCalledWith(file, 'content');
    expect(showToastMock).toHaveBeenCalledWith('articleEditor.contentImageUploadSuccess', 'success');
  });

  it('auto-saves a new draft article and redirects to the persisted edit page', async () => {
    vi.useFakeTimers();

    renderEditor();

    const textarea = screen.getByLabelText('content-editor');
    fireEvent.change(textarea, { target: { value: '新文章内容已经超过十个字，足够触发自动保存。' } });

    await Promise.resolve();
    await vi.advanceTimersByTimeAsync(30000);
    await Promise.resolve();

    expect(apiMocks.articleApi.createArticle).toHaveBeenCalledWith({
      title: 'articleEditor.noTitleDraft',
      summary: '',
      content: '新文章内容已经超过十个字，足够触发自动保存。',
      cover_image: '',
      category_id: undefined,
      status: 'draft',
      scheduled_publish_at: '',
      collection_id: undefined,
      collection_position: 0,
      tag_ids: [],
    });

    expect(navigateMock).toHaveBeenCalledWith('/admin/articles/edit/99', { replace: true });
  });
});
