import type { AIWritingAction } from '@/api/chat';

export type EditorMode = 'edit' | 'split' | 'preview';

export interface ContentStats {
  characters: number;
  lines: number;
  words: number;
  readingMinutes: number;
}

export interface SummaryPanelState {
  isGenerating: boolean;
  result: string;
}

export interface WritingPanelState {
  action: AIWritingAction;
  isGenerating: boolean;
  result: string;
  suggestions: string[];
}
