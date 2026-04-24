# AI RAG Fallback Design

## Overview
Improve the AI assistant so it no longer always replies with a fixed "not found in blog" message when vector retrieval returns no or weak matches. The system should prefer Redis vector search results when they are relevant, and fall back to a general-knowledge LLM answer when retrieval does not provide enough useful context.

## Goal
Make AI chat behave in two stages:
1. If Redis vector search returns sufficiently relevant blog content, use that content as grounded context for the answer.
2. If Redis returns no results or only weak results, let the LLM answer using general knowledge, while clearly telling the user that the answer is not based on site articles.

## Current Problem
The current implementation in `backend/internal/pkg/eino/chain.go` immediately returns a fixed fallback sentence when retrieval returns zero documents. It also uses a strict system prompt that forbids using any external knowledge. As a result:
- the assistant cannot continue answering when the knowledge base misses,
- the user experience is poor,
- the model never gets a chance to think independently.

There is also no usable similarity threshold today because `backend/internal/pkg/eino/vectorstore.go` does not return Redis vector scores in a way the chain can use for routing.

## Design Summary
We will implement a two-path answer flow inside the RAG chain:
- **RAG path**: used when retrieval returns documents whose top score is above a configurable threshold.
- **Fallback path**: used when retrieval returns no documents or only low-confidence matches.

This keeps business orchestration in `service/ai.go` simple and moves answer-mode selection into the Eino chain layer.

## Files Affected
- Modify: `backend/internal/pkg/eino/vectorstore.go`
- Modify: `backend/internal/pkg/eino/retriever.go`
- Modify: `backend/internal/pkg/eino/chain.go`
- Modify: `backend/internal/service/ai.go`
- Modify: `config/config.go`
- Modify: `config/config.yaml`

Optional tests, depending on current test layout:
- Create or modify tests around `backend/internal/pkg/eino/chain.go`
- Create or modify tests around `backend/internal/pkg/eino/retriever.go`

## Retrieval Decision Logic
The chain should follow this logic:

1. Embed the user question.
2. Query Redis vector search for topK results.
3. Read the highest similarity score from the search results.
4. Route to one of two paths:
   - **RAG path** if:
     - at least one document exists, and
     - the top score is greater than or equal to `RAGMinScore`
   - **Fallback path** otherwise

This means:
- `len(docs) == 0` is not a business failure.
- Low-score matches are treated the same as no match.
- Only embedding/search/generation infrastructure failures return real errors.

## Score Handling
`RedisVectorStore.Search` should return usable ranking information for routing. The current search command only returns fields like article_id, chunk_index, title, and content. The design changes are:

1. Update the Redis FT.SEARCH query to return the vector distance/score field.
2. Parse that field into `SearchResult.Score`.
3. Preserve the score when converting results into `schema.Document` by storing it in `MetaData`, for example:
   - `score`
   - `article_id`
   - `title`
   - `content`

The chain can then inspect the first document score without changing higher business layers.

## Configuration
Add a new AI config field:
- `RAGMinScore`

This should live alongside existing AI settings in `config/config.go` and `config/config.yaml`.

Initial recommended default: `0.30`

Reasoning:
- low enough to keep useful fuzzy matches,
- high enough to avoid noisy unrelated chunks,
- easy to tune later without code changes.

## Prompt Design
You asked specifically that prompts be more concrete. This design uses two separate, explicit prompts instead of one vague prompt.

### 1. RAG Prompt
Use this path only when retrieval confidence is high enough.

**Intent:** Answer from site content first, summarize clearly, and avoid inventing article claims.

**System prompt behavior:**
- You are the AI assistant for the 问道 blog.
- Your primary task is to answer based on the provided article excerpts.
- You may summarize, reorganize, and explain the excerpts in your own words.
- Do not claim the blog said something unless it is supported by the provided excerpts.
- If the excerpts only partially answer the question, explicitly say what is known from the site content and what is not covered.
- Use Chinese by default.
- Prefer a structured answer:
  - conclusion first,
  - then key points,
  - then a short supplement if necessary.

**Prompt skeleton:**
- Role and task definition
- Constraints on source fidelity
- Output format preference
- Context blocks labeled by excerpt number
- User question

### 2. Fallback Prompt
Use this path when retrieval confidence is too low.

**Intent:** Let the model answer helpfully using general knowledge, while making it explicit that the answer is not grounded in site articles.

**System prompt behavior:**
- You are the AI assistant for the 问道 blog.
- The current knowledge base search did not find enough relevant article content.
- Answer the user using general knowledge.
- The answer must begin with a clear note such as:
  - “以下回答基于通用知识，不是来自本站文章内容。”
- Do not pretend to cite or summarize blog articles.
- If the question is ambiguous, choose the most likely interpretation and state your assumption.
- If the question lacks important information, say what is missing.
- Be specific, practical, and avoid generic filler.
- Use Chinese.

### 3. Why Split Prompts
The grounded path and fallback path have different truth constraints:
- RAG mode must protect source fidelity.
- Fallback mode must protect attribution clarity.

Combining both into one broad prompt would make behavior unstable and blur source boundaries.

## Chain API Changes
`RAGChain.Execute` should keep the same external responsibility — produce an answer from a question — but internally it will:
- retrieve docs,
- inspect score,
- choose RAG or fallback message builder,
- call the LLM.

To keep the chain readable, split prompt creation into two dedicated helpers, for example:
- `buildRAGMessages(question, docs)`
- `buildFallbackMessages(question)`

This keeps prompt logic explicit and avoids one oversized conditional prompt builder.

## Error Handling
### These should use fallback, not return errors
- no retrieval results,
- low-confidence retrieval results.

### These should return real errors
- embedding failure,
- Redis FT.SEARCH failure,
- LLM generation failure.

This distinction is important: poor recall is a content condition, not a system failure.

## Logging
Keep logging concise. Add at most one debug/info-style event if needed in the AI flow to record which path was chosen:
- `mode=rag` with top score
- `mode=fallback` with top score or no results

Do not log full prompts or full user questions unless that is already an accepted pattern in the project.

## Testing Plan
Add tests for routing logic and prompt behavior.

Required cases:
1. **High-score retrieval**
   - documents exist,
   - top score >= threshold,
   - chain uses RAG prompt.
2. **No retrieval results**
   - chain uses fallback prompt,
   - no fixed "not found" message is returned.
3. **Low-score retrieval**
   - documents exist,
   - top score < threshold,
   - chain uses fallback prompt.
4. **Fallback answer attribution**
   - fallback response begins with a clear "general knowledge" statement.
5. **RAG answer attribution separation**
   - RAG prompt does not include fallback wording.
6. **Retriever/search failure**
   - chain returns an error.

## Backward Compatibility
This change does not alter the frontend request contract. The frontend still sends a question and receives an answer string. The change is purely in retrieval and prompt routing behavior.

## Recommended Implementation Order
1. Add score return support in `vectorstore.go`
2. Preserve score in `retriever.go`
3. Add threshold config in `config/config.go` and `config/config.yaml`
4. Refactor `chain.go` into RAG/fallback routing
5. Keep `service/ai.go` orchestration unchanged except for wiring any constructor changes
6. Add tests for routing and fallback

## Risks and Mitigations
### Risk: threshold too strict
If the threshold is too high, many valid questions will bypass RAG.

**Mitigation:** make the threshold configurable and start with a conservative default.

### Risk: threshold too loose
If the threshold is too low, bad excerpts will pollute answers.

**Mitigation:** use top-score routing and test with clearly unrelated queries.

### Risk: users confuse fallback answers with site content
**Mitigation:** force a clear opening sentence in fallback mode stating the answer is based on general knowledge.

## Acceptance Criteria
The design is successful when:
- the assistant no longer always returns the fixed "not found in blog" sentence,
- relevant Redis vector matches are still used as grounded context,
- empty or weak retrieval falls back to general-knowledge answering,
- fallback answers explicitly state they are not based on site articles,
- the routing threshold is configurable,
- frontend API behavior remains unchanged.
