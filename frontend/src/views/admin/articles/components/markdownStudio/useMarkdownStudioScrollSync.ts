import { type RefObject, useCallback, useEffect, useRef } from 'react';
import { getSyncedEditorScrollTop, getSyncedPreviewScrollTop } from './scrollSync';
import type { EditorMode } from './types';

interface UseMarkdownStudioScrollSyncOptions {
  content: string;
  editorMode: EditorMode;
  textareaRef: RefObject<HTMLTextAreaElement>;
  previewScrollRef: RefObject<HTMLDivElement>;
}

export const useMarkdownStudioScrollSync = ({
  content,
  editorMode,
  textareaRef,
  previewScrollRef,
}: UseMarkdownStudioScrollSyncOptions) => {
  const editorScrollMirrorRef = useRef<HTMLDivElement>(null);
  const scrollSyncFrameRef = useRef<number>();
  const isSyncingScrollRef = useRef(false);

  useEffect(() => {
    return () => {
      if (scrollSyncFrameRef.current) {
        cancelAnimationFrame(scrollSyncFrameRef.current);
      }
    };
  }, []);

  const syncMarkdownScroll = useCallback(
    (source: HTMLElement | null, target: HTMLElement | null) => {
      if (editorMode !== 'split' || !source || !target || isSyncingScrollRef.current) return;

      if (scrollSyncFrameRef.current) {
        cancelAnimationFrame(scrollSyncFrameRef.current);
      }

      scrollSyncFrameRef.current = requestAnimationFrame(() => {
        const editorMirror = editorScrollMirrorRef.current;
        const textarea = textareaRef.current;
        const preview = previewScrollRef.current;
        if (!editorMirror || !textarea || !preview) return;

        const nextScrollTop =
          source === textarea
            ? getSyncedPreviewScrollTop(textarea, preview, content, editorMirror)
            : getSyncedEditorScrollTop(preview, textarea, content, editorMirror);
        if (nextScrollTop === undefined) return;
        if (Math.abs(target.scrollTop - nextScrollTop) < 1) return;

        isSyncingScrollRef.current = true;
        target.scrollTop = nextScrollTop;

        requestAnimationFrame(() => {
          isSyncingScrollRef.current = false;
        });
      });
    },
    [content, editorMode, previewScrollRef, textareaRef]
  );

  const handleEditorScroll = useCallback(() => {
    syncMarkdownScroll(textareaRef.current, previewScrollRef.current);
  }, [previewScrollRef, syncMarkdownScroll, textareaRef]);

  const handlePreviewScroll = useCallback(() => {
    syncMarkdownScroll(previewScrollRef.current, textareaRef.current);
  }, [previewScrollRef, syncMarkdownScroll, textareaRef]);

  return {
    editorScrollMirrorRef,
    handleEditorScroll,
    handlePreviewScroll,
  };
};
