#!/usr/bin/env python3
import html
import re
import zipfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
MD_PATH = ROOT / "docs" / "信息工程_通信223_20221544309_李壮.md"
DOCX_PATH = ROOT / "docs" / "信息工程_通信223_20221544309_李壮_基于多Agent协作的个人博客智能问答系统.docx"

FONT_BODY = "仿宋"
FONT_COVER = "华文中宋"
FONT_HEADING = "仿宋"
FONT_ASCII = "Times New Roman"
LINE_EXACT_400 = 400


def esc(text: str) -> str:
    return html.escape(text, quote=False)


def rpr(font_size_half_points=24, bold=False, east_asia=FONT_BODY, ascii_font=FONT_ASCII):
    bold_xml = "<w:b/><w:bCs/>" if bold else ""
    return (
        "<w:rPr>"
        f"<w:rFonts w:ascii=\"{ascii_font}\" w:hAnsi=\"{ascii_font}\" w:eastAsia=\"{east_asia}\" w:cs=\"{ascii_font}\"/>"
        f"{bold_xml}"
        f"<w:sz w:val=\"{font_size_half_points}\"/><w:szCs w:val=\"{font_size_half_points}\"/>"
        "</w:rPr>"
    )


def run(text="", size=24, bold=False, font=FONT_BODY, ascii_font=FONT_ASCII, preserve=True):
    space = " xml:space=\"preserve\"" if preserve else ""
    return f"<w:r>{rpr(size, bold, font, ascii_font)}<w:t{space}>{esc(text)}</w:t></w:r>"


def br(kind="page"):
    return f"<w:r><w:br w:type=\"{kind}\"/></w:r>"


def p(
    text="",
    style=None,
    align=None,
    size=24,
    bold=False,
    first_line=False,
    spacing_after=0,
    line=LINE_EXACT_400,
    line_rule="exact",
    font=FONT_BODY,
    hanging=None,
):
    ppr = []
    if style:
        ppr.append(f"<w:pStyle w:val=\"{style}\"/>")
    if align:
        ppr.append(f"<w:jc w:val=\"{align}\"/>")
    ind_attrs = []
    if first_line:
        ind_attrs.append("w:firstLine=\"480\"")
    if hanging is not None:
        ind_attrs.append(f"w:left=\"{hanging}\" w:hanging=\"{hanging}\"")
    if ind_attrs:
        ppr.append(f"<w:ind {' '.join(ind_attrs)}/>")
    ppr.append(f"<w:spacing w:after=\"{spacing_after}\" w:line=\"{line}\" w:lineRule=\"{line_rule}\"/>")
    ppr_xml = f"<w:pPr>{''.join(ppr)}</w:pPr>"
    return f"<w:p>{ppr_xml}{run(text, size=size, bold=bold, font=font)}</w:p>"


def heading(text, level):
    if level == 1:
        return p(text, style="2", align="left", size=28, bold=True, spacing_after=0, font=FONT_HEADING)
    if level == 2:
        return p(text, style="3", align="left", size=24, bold=True, spacing_after=0, font=FONT_HEADING)
    return p(text, style="4", align="left", size=24, bold=True, spacing_after=0, font=FONT_HEADING)


def page_break():
    return f"<w:p>{br()}</w:p>"


def field_toc():
    return (
        "<w:p><w:pPr><w:jc w:val=\"center\"/><w:spacing w:after=\"0\" w:line=\"300\" w:lineRule=\"auto\"/></w:pPr>"
        f"{run('目  录', size=36, bold=True, font=FONT_BODY)}"
        "</w:p>"
        "<w:p>"
        "<w:r><w:fldChar w:fldCharType=\"begin\"/></w:r>"
        "<w:r><w:instrText xml:space=\"preserve\">TOC \\o \"1-3\" \\h \\z \\u</w:instrText></w:r>"
        "<w:r><w:fldChar w:fldCharType=\"separate\"/></w:r>"
        f"{run('右键目录区域，选择“更新域”后生成页码目录。', size=24, font=FONT_BODY)}"
        "<w:r><w:fldChar w:fldCharType=\"end\"/></w:r>"
        "</w:p>"
    )


def tc(text, width, align="center", bold=True, bottom=False):
    border = (
        "<w:tcBorders><w:bottom w:val=\"single\" w:sz=\"8\" w:space=\"0\" w:color=\"000000\"/></w:tcBorders>"
        if bottom
        else ""
    )
    return (
        f"<w:tc><w:tcPr><w:tcW w:w=\"{width}\" w:type=\"dxa\"/>{border}</w:tcPr>"
        f"{p(text, align=align, size=30, bold=bold, spacing_after=0, line=440, line_rule='exact', font=FONT_COVER)}"
        "</w:tc>"
    )


def cover_info_table(rows):
    tr_xml = []
    for label, value in rows:
        tr_xml.append(
            "<w:tr>"
            + tc(label, 3000, align="right", bold=True)
            + tc("", 420, align="center", bold=False)
            + tc(value, 3600, align="center", bold=True, bottom=True)
            + "</w:tr>"
        )
    return (
        "<w:tbl>"
        "<w:tblPr><w:tblW w:w=\"7020\" w:type=\"dxa\"/><w:jc w:val=\"center\"/>"
        "<w:tblBorders><w:top w:val=\"nil\"/><w:left w:val=\"nil\"/><w:bottom w:val=\"nil\"/>"
        "<w:right w:val=\"nil\"/><w:insideH w:val=\"nil\"/><w:insideV w:val=\"nil\"/></w:tblBorders>"
        "</w:tblPr>"
        "<w:tblGrid><w:gridCol w:w=\"3000\"/><w:gridCol w:w=\"420\"/><w:gridCol w:w=\"3600\"/></w:tblGrid>"
        + "".join(tr_xml)
        + "</w:tbl>"
    )


def table_cell(text, width, fill=None, bold=False, size=21, align="center", grid_span=None):
    props = [f"<w:tcW w:w=\"{width}\" w:type=\"dxa\"/>"]
    if grid_span:
        props.append(f"<w:gridSpan w:val=\"{grid_span}\"/>")
    props.append("<w:vAlign w:val=\"center\"/>")
    if fill:
        props.append(f"<w:shd w:val=\"clear\" w:color=\"auto\" w:fill=\"{fill}\"/>")
    paras = [
        p(part, align=align, size=size, bold=bold, spacing_after=0, line=300, line_rule="auto", font=FONT_BODY)
        for part in str(text).split("\n")
    ]
    return f"<w:tc><w:tcPr>{''.join(props)}</w:tcPr>{''.join(paras)}</w:tc>"


def word_table(rows, widths, header=True):
    grid = "".join(f"<w:gridCol w:w=\"{w}\"/>" for w in widths)
    tr_xml = []
    for row_index, row in enumerate(rows):
        cells = []
        for col_index, value in enumerate(row):
            fill = "D9EAF7" if header and row_index == 0 else ("F3F8FC" if row_index % 2 == 1 else None)
            cells.append(
                table_cell(
                    value,
                    widths[min(col_index, len(widths) - 1)],
                    fill=fill,
                    bold=header and row_index == 0,
                    size=21,
                    align="center" if row_index == 0 else "left",
                )
            )
        tr_xml.append("<w:tr>" + "".join(cells) + "</w:tr>")
    return (
        "<w:tbl>"
        "<w:tblPr><w:tblW w:w=\"0\" w:type=\"auto\"/><w:jc w:val=\"center\"/>"
        "<w:tblBorders><w:top w:val=\"single\" w:sz=\"6\" w:color=\"000000\"/>"
        "<w:left w:val=\"single\" w:sz=\"6\" w:color=\"000000\"/>"
        "<w:bottom w:val=\"single\" w:sz=\"6\" w:color=\"000000\"/>"
        "<w:right w:val=\"single\" w:sz=\"6\" w:color=\"000000\"/>"
        "<w:insideH w:val=\"single\" w:sz=\"4\" w:color=\"666666\"/>"
        "<w:insideV w:val=\"single\" w:sz=\"4\" w:color=\"666666\"/></w:tblBorders>"
        "<w:tblCellMar><w:top w:w=\"80\" w:type=\"dxa\"/><w:left w:w=\"80\" w:type=\"dxa\"/>"
        "<w:bottom w:w=\"80\" w:type=\"dxa\"/><w:right w:w=\"80\" w:type=\"dxa\"/></w:tblCellMar>"
        "</w:tblPr>"
        f"<w:tblGrid>{grid}</w:tblGrid>"
        + "".join(tr_xml)
        + "</w:tbl>"
    )


def caption(text):
    return p(text, align="center", size=21, spacing_after=0, line=300, line_rule="auto", font=FONT_BODY)


def figure_box(text, fill="EAF3F8", width=9000):
    return word_table([[text]], [width], header=False).replace("<w:tblBorders>", f"<w:tblLook w:val=\"04A0\"/><w:tblBorders>", 1).replace(
        "<w:tcPr><w:tcW", f"<w:tcPr><w:shd w:val=\"clear\" w:color=\"auto\" w:fill=\"{fill}\"/><w:tcW", 1
    )


def architecture_diagram():
    rows = [
        ["用户访问层", "浏览器 / 移动端 Web"],
        ["前端展示层", "React SPA、AI 聊天页、博客首页、后台管理"],
        ["接口服务层", "Gin Router、JWT 鉴权、CORS、中间件、SSE 流式响应"],
        ["业务服务层", "文章服务、上传服务、统计服务、知识文档服务、ThinkTank 多 Agent 服务"],
        ["智能能力层", "LocalSearch、WebSearch、WebFetch、DocWriter、Synthesizer"],
        ["数据存储层", "MySQL、Redis、Redis Vector、uploads 文件目录、外部大模型服务"],
    ]
    return word_table(rows, [2200, 7000], header=False) + caption("图4-1 系统总体架构图")


def module_diagram():
    rows = [
        ["前台博客模块", "首页列表、文章详情、分类浏览、评论展示"],
        ["用户认证模块", "注册登录、JWT 鉴权、GitHub OAuth、头像维护"],
        ["后台管理模块", "文章管理、分类管理、评论管理、数据统计、知识文档审核"],
        ["上传与水印模块", "封面图上传、正文图片上传、头像上传、原创图片水印"],
        ["AI 问答模块", "会话管理、RAG 检索、多 Agent 编排、流式响应"],
        ["知识文档模块", "调研文档生成、审核、首页文章化、向量化入库"],
    ]
    return word_table(rows, [2600, 6600], header=False) + caption("图4-2 系统功能模块图")


def agent_flow_diagram():
    rows = [
        ["1", "用户问题", "提交自然语言问题，并携带当前会话上下文"],
        ["2", "Planner", "判断是否需要澄清，规划执行路径"],
        ["3", "Librarian", "调用 LocalSearch 检索站内文章和知识文档"],
        ["4", "Journalist", "在本地知识不足时执行 WebSearch 和 WebFetch"],
        ["5", "DocWriter", "将有价值的调研结果保存为候选知识文档"],
        ["6", "Synthesizer", "整合本地检索、外部调研和引用来源，生成最终回答"],
        ["7", "过程记录", "将 Agent 阶段、工具调用和返回结果写入 MySQL"],
    ]
    return word_table(rows, [900, 2200, 6100]) + caption("图4-3 多 Agent 智能问答流程图")


def knowledge_loop_diagram():
    rows = [
        ["用户提问", "站内向量检索", "外部调研"],
        ["生成最终回答", "生成知识文档草稿", "管理员审核"],
        ["首页文章化展示", "向量化入库", "后续问答复用"],
    ]
    return word_table(rows, [3000, 3000, 3000], header=False) + caption("图4-4 问答驱动的知识沉淀闭环图")


DB_TABLES = {
    "5.1 用户与权限模型": (
        "表5-1 用户表主要字段",
        [
            ["字段名", "类型", "约束", "说明"],
            ["id", "bigint", "主键", "用户唯一标识"],
            ["username", "varchar", "唯一", "用户登录名"],
            ["email", "varchar", "唯一", "用户邮箱"],
            ["password_hash", "varchar", "非空", "密码哈希值"],
            ["role", "varchar", "默认 user", "用户角色，区分普通用户和管理员"],
            ["avatar_url", "varchar", "可空", "用户头像地址"],
            ["oauth_provider", "varchar", "可空", "第三方登录来源"],
        ],
    ),
    "5.2 文章与评论模型": (
        "表5-2 文章表主要字段",
        [
            ["字段名", "类型", "约束", "说明"],
            ["id", "bigint", "主键", "文章唯一标识"],
            ["title", "varchar", "非空", "文章标题"],
            ["summary", "text", "可空", "文章摘要"],
            ["content", "longtext", "非空", "Markdown 正文"],
            ["status", "varchar", "索引", "草稿或已发布状态"],
            ["source_type", "varchar", "索引", "区分普通文章和知识文档文章"],
            ["source_id", "bigint", "可空", "关联知识文档 ID"],
            ["ai_indexed_at", "datetime", "可空", "向量化完成时间"],
        ],
    ),
    "5.3 AI 会话与记忆模型": (
        "表5-3 AI 会话与记忆相关表",
        [
            ["表名", "核心字段", "说明"],
            ["conversations", "id、user_id、title、created_at", "保存一次 AI 聊天会话"],
            ["chat_messages", "conversation_id、role、content、metadata", "保存用户消息和 AI 回复"],
            ["conversation_memories", "conversation_id、memory_type、summary", "保存长期记忆摘要和项目事实"],
        ],
    ),
    "5.4 多 Agent 执行过程模型": (
        "表5-4 多 Agent 执行过程表",
        [
            ["表名", "核心字段", "说明"],
            ["conversation_runs", "conversation_id、status、started_at、finished_at", "记录一次多 Agent 执行"],
            ["conversation_run_steps", "run_id、stage、message、detail、metadata", "记录 Agent 阶段、工具调用和返回结果"],
        ],
    ),
    "5.5 知识文档与引用来源模型": (
        "表5-5 知识文档与引用来源表",
        [
            ["表名", "核心字段", "说明"],
            ["knowledge_documents", "title、summary、content、status、article_id、vectorized_at", "保存调研生成的知识文档"],
            ["knowledge_document_sources", "document_id、title、url、summary", "保存外部参考来源"],
            ["article_vectors", "source_type、source_id、chunk_index、embedding", "保存文章和知识文档分块向量"],
        ],
    ),
}


def data_table_for_heading(title):
    if title not in DB_TABLES:
        return ""
    caption_text, rows = DB_TABLES[title]
    width_count = len(rows[0])
    widths = [1900, 2200, 1900, 3300] if width_count == 4 else [2500, 3600, 3100]
    return word_table(rows, widths, header=True) + caption(caption_text)


def cover_page():
    parts = []
    parts.append(p("河南科技学院", align="center", size=36, bold=True, font=FONT_COVER, spacing_after=260, line=440, line_rule="exact"))
    parts.append(p("2026届本科毕业论文（设计）", align="center", size=36, bold=True, font=FONT_COVER, spacing_after=780, line=440, line_rule="exact"))
    parts.append(p("基于多 Agent 协作的", align="center", size=36, bold=True, font=FONT_COVER, spacing_after=0, line=400, line_rule="exact"))
    parts.append(p("个人博客智能问答系统", align="center", size=36, bold=True, font=FONT_COVER, spacing_after=820, line=400, line_rule="exact"))
    rows = [
        ("学生学号：", "__________"),
        ("学生姓名：", "李壮"),
        ("所在学院：", "信息工程学院"),
        ("所学专业：", "信息工程"),
        ("导师姓名：", "__________"),
        ("完成时间：", "2026年5月"),
    ]
    parts.append(cover_info_table(rows))
    parts.append(page_break())
    return "".join(parts)


def normalize_line(line: str) -> str:
    line = line.strip()
    line = re.sub(r"^\*\*(.+?)\*\*$", r"\1", line)
    line = line.replace("**", "")
    return line


def md_to_body(md: str):
    lines = md.splitlines()
    out = [cover_page()]
    in_code = False
    skip_manual_toc = False
    started = False
    abstract_mode = None

    for raw in lines:
        line = normalize_line(raw)
        if not line:
            continue
        if line.startswith("# 基于多 Agent"):
            continue
        if line.startswith("学生学号") or line.startswith("学生姓名") or line.startswith("所在学院") or line.startswith("所学专业") or line.startswith("导师姓名") or line.startswith("完成时间"):
            continue
        if line == "---":
            continue
        if line.startswith("```"):
            in_code = not in_code
            continue
        if line == "## 目  录":
            abstract_mode = None
            out.append(page_break())
            out.append(field_toc())
            out.append(page_break())
            skip_manual_toc = True
            continue
        if skip_manual_toc:
            if line.startswith("## 1 绪论"):
                skip_manual_toc = False
            else:
                continue
        if line.startswith("> 注："):
            out.append(p(line.lstrip("> "), size=21, first_line=False, spacing_after=0))
            continue
        if line.startswith("|"):
            out.append(p(line, size=21, first_line=False, spacing_after=0))
            continue
        if in_code:
            continue
        if line.startswith("## "):
            title = line[3:]
            if title == "摘  要":
                abstract_mode = "cn"
                out.append(p(title, align="center", size=28, bold=True, spacing_after=0, font=FONT_BODY))
                continue
            if title == "Abstract":
                abstract_mode = "en"
                out.append(page_break())
                out.append(
                    p(
                        "Personal Blog Intelligent Question Answering System Based on Multi-Agent Collaboration",
                        align="center",
                        size=36,
                        bold=True,
                        spacing_after=0,
                        font=FONT_ASCII,
                    )
                )
                out.append(p(title, align="center", size=28, bold=True, spacing_after=0, font=FONT_ASCII))
                continue
            abstract_mode = None
            if started and re.match(r"^[1-8] ", title):
                out.append(page_break())
            started = True
            out.append(heading(title, 1))
            continue
        if line.startswith("### "):
            title = line[4:]
            out.append(heading(title, 2))
            if title == "4.1 系统总体架构":
                out.append(architecture_diagram())
            elif title == "4.2 系统功能模块划分":
                out.append(module_diagram())
            elif title == "4.3 多 Agent 智能问答流程设计":
                out.append(agent_flow_diagram())
            elif title == "4.4 问答驱动的知识沉淀闭环设计":
                out.append(knowledge_loop_diagram())
            else:
                out.append(data_table_for_heading(title))
            continue
        if line.startswith("#### "):
            out.append(heading(line[5:], 3))
            continue
        if re.match(r"^\[\d+\]", line):
            out.append(p(line, size=21, first_line=False, spacing_after=0, hanging=420))
            continue
        if line.startswith("- "):
            out.append(p("• " + line[2:], size=24, first_line=False, spacing_after=0))
            continue
        if re.match(r"^\d+ ", line):
            out.append(p(line, size=24, first_line=False, spacing_after=0))
            continue
        if line.startswith("关键词") or line.startswith("Key Words"):
            out.append(p(line, size=24, bold=True, first_line=False, spacing_after=0, font=FONT_BODY if line.startswith("关键词") else FONT_ASCII))
            continue
        if abstract_mode == "en":
            out.append(p(line, size=24, first_line=True, spacing_after=0, font=FONT_ASCII, align="both", line=300, line_rule="auto"))
        else:
            out.append(p(line, size=24, first_line=True, spacing_after=0, font=FONT_BODY, align="both"))
    return "".join(out)


def styles_xml():
    return """<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:styles xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:docDefaults>
    <w:rPrDefault><w:rPr><w:rFonts w:ascii="Times New Roman" w:hAnsi="Times New Roman" w:eastAsia="仿宋" w:cs="Times New Roman"/><w:sz w:val="24"/><w:szCs w:val="24"/></w:rPr></w:rPrDefault>
    <w:pPrDefault><w:pPr><w:spacing w:after="0" w:line="400" w:lineRule="exact"/></w:pPr></w:pPrDefault>
  </w:docDefaults>
  <w:style w:type="paragraph" w:default="1" w:styleId="1"><w:name w:val="Normal"/><w:qFormat/><w:rPr><w:rFonts w:ascii="Times New Roman" w:hAnsi="Times New Roman" w:eastAsia="仿宋" w:cs="Times New Roman"/><w:sz w:val="24"/><w:szCs w:val="24"/></w:rPr></w:style>
  <w:style w:type="paragraph" w:styleId="2"><w:name w:val="heading 1"/><w:basedOn w:val="1"/><w:next w:val="1"/><w:uiPriority w:val="9"/><w:qFormat/><w:pPr><w:spacing w:after="0" w:line="400" w:lineRule="exact"/><w:outlineLvl w:val="0"/></w:pPr><w:rPr><w:b/><w:bCs/><w:rFonts w:eastAsia="仿宋" w:ascii="Times New Roman" w:hAnsi="Times New Roman"/><w:sz w:val="28"/><w:szCs w:val="28"/></w:rPr></w:style>
  <w:style w:type="paragraph" w:styleId="3"><w:name w:val="heading 2"/><w:basedOn w:val="1"/><w:next w:val="1"/><w:uiPriority w:val="9"/><w:qFormat/><w:pPr><w:spacing w:after="0" w:line="400" w:lineRule="exact"/><w:outlineLvl w:val="1"/></w:pPr><w:rPr><w:b/><w:bCs/><w:rFonts w:eastAsia="仿宋" w:ascii="Times New Roman" w:hAnsi="Times New Roman"/><w:sz w:val="24"/><w:szCs w:val="24"/></w:rPr></w:style>
  <w:style w:type="paragraph" w:styleId="4"><w:name w:val="heading 3"/><w:basedOn w:val="1"/><w:next w:val="1"/><w:uiPriority w:val="9"/><w:qFormat/><w:pPr><w:spacing w:after="0" w:line="400" w:lineRule="exact"/><w:outlineLvl w:val="2"/></w:pPr><w:rPr><w:b/><w:bCs/><w:rFonts w:eastAsia="仿宋" w:ascii="Times New Roman" w:hAnsi="Times New Roman"/><w:sz w:val="24"/><w:szCs w:val="24"/></w:rPr></w:style>
  <w:style w:type="paragraph" w:styleId="16"><w:name w:val="toc 1"/><w:basedOn w:val="1"/><w:pPr><w:tabs><w:tab w:val="right" w:leader="dot" w:pos="9350"/></w:tabs><w:spacing w:after="0" w:line="300" w:lineRule="auto"/></w:pPr><w:rPr><w:rFonts w:ascii="Times New Roman" w:hAnsi="Times New Roman" w:eastAsia="仿宋"/><w:sz w:val="24"/><w:szCs w:val="24"/></w:rPr></w:style>
  <w:style w:type="paragraph" w:styleId="18"><w:name w:val="toc 2"/><w:basedOn w:val="16"/><w:pPr><w:tabs><w:tab w:val="right" w:leader="dot" w:pos="9350"/></w:tabs><w:ind w:left="480"/><w:spacing w:after="0" w:line="300" w:lineRule="auto"/></w:pPr></w:style>
  <w:style w:type="paragraph" w:styleId="10"><w:name w:val="toc 3"/><w:basedOn w:val="16"/><w:pPr><w:tabs><w:tab w:val="right" w:leader="dot" w:pos="9350"/></w:tabs><w:ind w:left="960"/><w:spacing w:after="0" w:line="300" w:lineRule="auto"/></w:pPr></w:style>
</w:styles>"""


def document_xml(body: str):
    return f"""<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships">
  <w:body>
    {body}
    <w:sectPr>
      <w:pgSz w:w="11907" w:h="16840"/>
      <w:pgMar w:top="1417" w:right="1417" w:bottom="1417" w:left="1417" w:header="850" w:footer="992" w:gutter="0"/>
      <w:cols w:space="425"/>
      <w:docGrid w:type="lines" w:linePitch="312"/>
    </w:sectPr>
  </w:body>
</w:document>"""


def write_docx():
    md = MD_PATH.read_text(encoding="utf-8")
    body = md_to_body(md)
    content_types = """<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
  <Override PartName="/word/styles.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.styles+xml"/>
  <Override PartName="/word/settings.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.settings+xml"/>
</Types>"""
    rels = """<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>"""
    doc_rels = """<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"/>"""
    settings = """<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:settings xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:updateFields w:val="true"/>
  <w:defaultTabStop w:val="420"/>
  <w:characterSpacingControl w:val="compressPunctuation"/>
</w:settings>"""
    with zipfile.ZipFile(DOCX_PATH, "w", zipfile.ZIP_DEFLATED) as z:
        z.writestr("[Content_Types].xml", content_types)
        z.writestr("_rels/.rels", rels)
        z.writestr("word/_rels/document.xml.rels", doc_rels)
        z.writestr("word/document.xml", document_xml(body))
        z.writestr("word/styles.xml", styles_xml())
        z.writestr("word/settings.xml", settings)
    print(DOCX_PATH)


if __name__ == "__main__":
    write_docx()
