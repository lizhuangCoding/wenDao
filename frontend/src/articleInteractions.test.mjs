import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';

const loadSource = async (path) => readFile(new URL(path, import.meta.url), 'utf8');

test('article API exposes authenticated like favorite and personal list endpoints', async () => {
  const source = await loadSource('./api/article.ts');

  assert.match(source, /getArticleInteraction/);
  assert.match(source, /likeArticle/);
  assert.match(source, /unlikeArticle/);
  assert.match(source, /favoriteArticle/);
  assert.match(source, /unfavoriteArticle/);
  assert.match(source, /getLikedArticles/);
  assert.match(source, /getFavoriteArticles/);
  assert.match(source, /\/articles\/\$\{id\}\/interaction/);
  assert.match(source, /\/users\/me\/liked-articles/);
  assert.match(source, /\/users\/me\/favorite-articles/);
});

test('article detail renders like and favorite actions using authenticated interaction state', async () => {
  const source = await loadSource('./pages/ArticleDetail.tsx');

  assert.match(source, /Heart/);
  assert.match(source, /Bookmark/);
  assert.match(source, /getArticleInteraction/);
  assert.match(source, /likeArticle/);
  assert.match(source, /unlikeArticle/);
  assert.match(source, /favoriteArticle/);
  assert.match(source, /unfavoriteArticle/);
  assert.match(source, /interactionQuery/);
  assert.match(source, /t\('article\.like'\)/);
  assert.match(source, /t\('article\.favorite'\)/);
});

test('article detail places like and favorite actions after the article body before comments', async () => {
  const source = await loadSource('./pages/ArticleDetail.tsx');

  const contentIndex = source.indexOf('<ArticleContent content={article.content} />');
  const actionIndex = source.indexOf('article-interaction-actions');
  const commentsIndex = source.indexOf('<CommentList articleId={article.id} totalCommentCount={article.comment_count} />');

  assert.notEqual(contentIndex, -1);
  assert.notEqual(actionIndex, -1);
  assert.notEqual(commentsIndex, -1);
  assert.ok(contentIndex < actionIndex, 'expected interaction actions after article content');
  assert.ok(actionIndex < commentsIndex, 'expected interaction actions before comments');
});

test('profile page exposes liked and favorite article tabs', async () => {
  const source = await loadSource('./pages/Profile.tsx');

  assert.match(source, /getLikedArticles/);
  assert.match(source, /getFavoriteArticles/);
  assert.match(source, /liked-articles/);
  assert.match(source, /favorite-articles/);
  assert.match(source, /t\('profile\.likedArticles'\)/);
  assert.match(source, /t\('profile\.favoriteArticles'\)/);
  assert.match(source, /ArticleCard/);
});
