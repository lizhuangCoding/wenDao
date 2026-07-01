export type ChatStage =
  | 'analyzing'
  | 'clarifying_intent'
  | 'clarifying'
  | 'adk_event'
  | 'local_search'
  | 'web_research'
  | 'integration'
  | 'synthesizing'
  | 'reviewing'
  | 'revising'
  | 'streaming'
  | 'completed'
  | 'failed';

export interface ChatStageEvent {
  stage: ChatStage;
  label?: string;
}

export interface ChatStep {
  id: number;
  run_id?: number;
  agent_name: string;
  type: string;
  summary: string;
  detail: string;
  status: 'running' | 'completed' | 'failed';
  created_at: string;
}

export interface ChatStepEvent {
  run_id?: number;
  step_id: number;
  agent_name: string;
  status: 'running' | 'completed' | 'failed';
  summary: string;
  detail: string;
}

export interface ChatActiveRun {
  id: number;
  status: 'idle' | 'running' | 'waiting_user' | 'completed' | 'failed';
  current_stage: ChatStage | 'streaming';
  pending_question?: string;
  last_answer: string;
  heartbeat_at?: string;
  can_resume: boolean;
}

export interface ChatResumeEvent {
  run_id: number;
  stage?: ChatStage | 'streaming';
  status?: ChatActiveRun['status'];
}

export interface ChatSnapshotEvent {
  run_id: number;
  stage?: ChatStage | 'streaming';
  status?: ChatActiveRun['status'];
  message?: string;
}

export interface ChatHeartbeatEvent {
  run_id: number;
  stage?: ChatStage | 'streaming';
  status?: ChatActiveRun['status'];
}

export interface ChatArticleReference {
  title: string;
  url: string;
}

export interface ChatReferenceGroups {
  blog: ChatArticleReference[];
  external: ChatArticleReference[];
}

export interface ChatQuestionEvent {
  run_id?: number;
  stage: 'clarifying';
  message: string;
  requires_user_input: true;
}

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: number;
  processSteps?: ChatStep[];
  runId?: number;
}

export interface ChatRequest {
  message: string;
  article_id?: number;
  conversation_id?: number;
  model_provider?: string;
  model_name?: string;
}

export interface ModelInfo {
  provider: string;
  model_name: string;
  display_name: string;
}

export interface ChatResponse {
  message: string;
  sources?: string[];
}

export interface SharedConversationData {
  conversation: {
    id: number;
    user_id: number;
    title: string;
    is_shared: boolean;
    share_token: string;
    created_at: string;
    updated_at: string;
  };
  messages: Array<{
    id: number;
    conversation_id: number;
    run_id?: number;
    role: 'user' | 'assistant';
    content: string;
    created_at: string;
    process_steps?: ChatStep[];
  }>;
  steps?: ChatStep[];
  shared_by: {
    username: string;
    avatar_url?: string;
  };
}

export interface ChatConversationSummary {
  id: number;
  title: string;
  user_id: number;
  is_shared?: boolean;
  share_token?: string;
  created_at: string;
  updated_at: string;
}

export interface ChatConversationMutationResponse {
  id: number;
  title: string;
  user_id: number;
  is_shared?: boolean;
  share_token?: string;
  created_at: string;
  updated_at: string;
}

export interface ChatConversationDetailResponse {
  conversation: {
    id: number;
    title: string;
    user_id: number;
    is_shared: boolean;
    share_token?: string;
    created_at: string;
    updated_at: string;
  };
  messages: Array<{
    id: number;
    conversation_id: number;
    run_id?: number;
    role: 'user' | 'assistant';
    content: string;
    created_at: string;
    process_steps?: ChatStep[];
  }>;
  steps?: ChatStep[];
  active_run?: ChatActiveRun;
  active_steps?: ChatStep[];
}

export interface AIObservabilityToolUsage {
  local_search: number;
  web_search: number;
  web_fetch: number;
  doc_writer: number;
  other: number;
}

export interface AIObservabilitySources {
  local_hits: number;
  web_hits: number;
  quality_score: number;
  external_urls: Array<{
    url: string;
    quality_score: number;
  }>;
}

export interface AIObservabilityCost {
  status: 'not_collected' | 'tokens_only' | 'estimated' | 'collected';
  prompt_tokens: number;
  completion_tokens: number;
  estimated_cost: number;
  currency: string;
}

export interface AIObservabilityFeedback {
  status: 'not_collected' | 'collected';
  score?: number;
}

export interface AIObservabilityFailedStep {
  id: number;
  agent_name: string;
  type: string;
  summary: string;
  detail: string;
  category: string;
  created_at: string;
}

export interface AIObservabilityFailureCluster {
  category: string;
  count: number;
}

export interface AIObservabilityStep {
  id: number;
  agent_name: string;
  type: string;
  summary: string;
  status: string;
  created_at: string;
}

export interface AIObservabilityRun {
  id: number;
  conversation_id: number;
  user_id: number;
  status: string;
  current_stage: string;
  original_question: string;
  normalized_question: string;
  last_error?: string;
  duration_seconds: number;
  step_count: number;
  failed_step_count: number;
  tool_usage: AIObservabilityToolUsage;
  sources: AIObservabilitySources;
  cost: AIObservabilityCost;
  feedback: AIObservabilityFeedback;
  failure_category?: string;
  failure_fingerprint?: string;
  failed_steps: AIObservabilityFailedStep[];
  failure_clusters: AIObservabilityFailureCluster[];
  steps: AIObservabilityStep[];
  created_at: string;
  updated_at: string;
  completed_at?: string;
  heartbeat_at?: string;
}
