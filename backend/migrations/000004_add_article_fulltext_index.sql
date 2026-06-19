ALTER TABLE articles
  ADD FULLTEXT KEY ft_articles_search (title, summary, content);
