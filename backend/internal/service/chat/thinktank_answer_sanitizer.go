package chat

import "strings"

func sanitizeFinalAnswerForUser(answer string) string {
	paragraphs := splitAnswerParagraphs(answer)
	if len(paragraphs) == 0 {
		return strings.TrimSpace(answer)
	}

	cleaned := make([]string, 0, len(paragraphs))
	for i := 0; i < len(paragraphs); i++ {
		paragraph := strings.TrimSpace(paragraphs[i])
		if paragraph == "" {
			continue
		}
		if isReferenceParagraph(paragraph) {
			cleaned = append(cleaned, paragraph)
			continue
		}
		if isRuntimeLimitationHeadingOnly(paragraph) && i+1 < len(paragraphs) && containsRuntimeFailureDetail(paragraphs[i+1]) {
			i++
			continue
		}
		if startsWithRuntimeLimitationHeading(paragraph) && containsRuntimeFailureDetail(paragraph) {
			continue
		}
		if containsAnswerProcessSummary(paragraph) {
			continue
		}
		if containsDocWriterMetadata(paragraph) {
			continue
		}
		if containsToolInputRequestToUser(paragraph) {
			continue
		}
		paragraph = removeRuntimeFailureLines(paragraph)
		if strings.TrimSpace(paragraph) != "" {
			cleaned = append(cleaned, strings.TrimSpace(paragraph))
		}
	}
	return strings.Join(cleaned, "\n\n")
}

func splitAnswerParagraphs(answer string) []string {
	lines := strings.Split(strings.ReplaceAll(answer, "\r\n", "\n"), "\n")
	paragraphs := make([]string, 0)
	current := make([]string, 0)
	flush := func() {
		if len(current) == 0 {
			return
		}
		paragraphs = append(paragraphs, strings.Join(current, "\n"))
		current = current[:0]
	}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		current = append(current, line)
	}
	flush()
	return paragraphs
}

func removeRuntimeFailureLines(paragraph string) string {
	lines := strings.Split(paragraph, "\n")
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		if isReferenceLine(line) || (!containsRuntimeFailureDetail(line) && !containsAnswerProcessSummary(line) && !containsDocWriterMetadata(line) && !containsToolInputRequestToUser(line)) {
			kept = append(kept, line)
		}
	}
	return strings.Join(kept, "\n")
}

func isReferenceParagraph(paragraph string) bool {
	first := normalizedAnswerHeading(firstNonEmptyLine(paragraph))
	return strings.Contains(first, "参考博主文章") || strings.Contains(first, "参考外部文章")
}

func isReferenceLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "- [") || strings.HasPrefix(trimmed, "* [") || strings.HasPrefix(trimmed, "[")
}

func isRuntimeLimitationHeadingOnly(paragraph string) bool {
	lines := strings.Split(strings.TrimSpace(paragraph), "\n")
	if len(lines) != 1 {
		return false
	}
	return isRuntimeLimitationHeading(lines[0])
}

func startsWithRuntimeLimitationHeading(paragraph string) bool {
	return isRuntimeLimitationHeading(firstNonEmptyLine(paragraph))
}

func isRuntimeLimitationHeading(line string) bool {
	heading := normalizedAnswerHeading(line)
	return strings.HasPrefix(heading, "证据局限") ||
		strings.HasPrefix(heading, "证据限制") ||
		strings.HasPrefix(heading, "回答限制") ||
		strings.HasPrefix(heading, "工具限制")
}

func normalizedAnswerHeading(line string) string {
	heading := strings.TrimSpace(line)
	heading = strings.TrimLeft(heading, "#*-= ")
	heading = strings.TrimSpace(heading)
	heading = strings.TrimRight(heading, "：: ")
	return heading
}

func firstNonEmptyLine(paragraph string) string {
	for _, line := range strings.Split(paragraph, "\n") {
		if strings.TrimSpace(line) != "" {
			return line
		}
	}
	return ""
}

func containsRuntimeFailureDetail(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	markers := []string{
		"返回状态码",
		"状态码 404",
		"状态码404",
		"未搜索到",
		"没有搜索到",
		"未能成功抓取",
		"抓取失败",
		"网页抓取",
		"webfetch",
		"websearch",
		"localsearch",
		"工具失败",
		"工具错误",
		"工具报错",
		"工具调用失败",
		"工具不可用",
		"工具限制",
		"候选页面",
		"不是 url",
		"不是url",
		"网络原因",
		"upstream",
		"timeout",
		"超时",
		"无法访问",
		"不可用页面",
		"本站没有覆盖",
		"本站没有提供",
		"无法根据文章片段回答",
	}
	for _, marker := range markers {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	if strings.Contains(normalized, "404") && containsAny(normalized, []string{"网页", "页面", "抓取", "返回", "状态码", "百度百科"}) {
		return true
	}
	if strings.Contains(normalized, "未配置") &&
		containsAny(normalized, []string{"联网调研", "本地知识库", "websearch", "webfetch", "localsearch", "工具", "服务", "api", "提供商"}) {
		return true
	}
	if (strings.Contains(normalized, "未找到") || strings.Contains(normalized, "没有找到")) &&
		containsAny(normalized, []string{"搜索", "网页", "页面", "来源", "资料", "证据", "结果", "websearch", "localsearch"}) {
		return true
	}
	return false
}

func containsAnswerProcessSummary(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	heading := normalizedAnswerHeading(firstNonEmptyLine(text))
	if strings.HasPrefix(heading, "验收摘要") {
		return true
	}
	if containsAny(normalized, []string{
		"通过对站内知识",
		"通过站内知识",
		"站内知识库和网络搜索",
		"站内知识和网络搜索",
		"站内知识库查询",
		"网络搜索信息的整合",
	}) {
		return true
	}
	if containsAny(normalized, []string{"综上所述", "总之", "整体来看"}) &&
		containsAny(normalized, []string{"站内知识", "知识库", "网络搜索", "完成了", "满足了"}) {
		return true
	}
	if containsAny(normalized, []string{"完成了对", "完成对"}) &&
		containsAny(normalized, []string{"调研", "研究", "分析"}) &&
		containsAny(normalized, []string{"本次", "此次", "目标", "需求"}) {
		return true
	}
	if containsAny(normalized, []string{"满足了对", "满足用户", "满足了用户"}) &&
		containsAny(normalized, []string{"目标", "需求", "问题"}) {
		return true
	}
	if containsAny(normalized, []string{
		"本次调研全面涵盖",
		"本次回答全面涵盖",
		"本次分析全面涵盖",
	}) {
		return true
	}
	if containsAny(normalized, []string{"已为你调研到", "已为你搜索到", "已为你找到", "调研到"}) &&
		containsAny(normalized, []string{"后续可根据", "目标已完成", "执行过程中", "多渠道查询", "相关网页链接"}) {
		return true
	}
	if containsAny(normalized, []string{"目标已完成", "执行过程中"}) &&
		containsAny(normalized, []string{"检索本地知识库", "搜索引擎", "多渠道查询", "成功获取到了所需的信息"}) {
		return true
	}
	return false
}

func containsDocWriterMetadata(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	if containsAny(normalized, []string{
		"docwriter",
		"doc_id",
		"document id",
		"draft id",
		"知识文档草稿",
		"文档草稿",
		"保存为知识文档",
		"成功创建知识文档",
		"调研文档存盘",
		"文档 id",
		"文档id",
	}) {
		return true
	}
	if strings.Contains(normalized, "id=") &&
		containsAny(normalized, []string{"文档", "草稿", "draft", "doc"}) {
		return true
	}
	return false
}

func containsToolInputRequestToUser(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	if containsAny(normalized, []string{"websearch", "webfetch", "localsearch"}) &&
		containsAny(normalized, []string{"请提供", "提供", "链接", "url", "网页"}) {
		return true
	}
	if containsAny(normalized, []string{"我将获取这些网页", "将获取这些网页", "获取这些网页的详细内容"}) {
		return true
	}
	if containsAny(normalized, []string{"请提供"}) &&
		containsAny(normalized, []string{"相关新闻链接", "网页链接", "具体链接"}) &&
		containsAny(normalized, []string{"获取", "抓取", "详细内容"}) {
		return true
	}
	return false
}

func containsAny(text string, needles []string) bool {
	for _, needle := range needles {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}
