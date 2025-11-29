package ssrn

import (
	"regexp"
	"strings"
)

// 提取搜索结果中的 abstract_id 列表
var (
	reAbsLink     = regexp.MustCompile(`href="https://papers\.ssrn\.com/sol3/papers\.cfm\?abstract_id=([0-9]+)`)          // 匹配所有链接
	reTitle       = regexp.MustCompile(`(?is)<title>([^<]+)</title>`)                                                     // 标题
	reAbsBox      = regexp.MustCompile(`(?is)<div[^>]*class=["']?[^>"']*abstract-text[^>"']*["']?[^>]*>([\s\S]*?)</div>`) // 摘要容器
	reTags        = regexp.MustCompile(`(?is)<[^>]+>`)                                                                    // 去标签
	reCanonical   = regexp.MustCompile(`(?is)<link[^>]*rel=\"canonical\"[^>]*href=\"([^\"]+)\"`)                          // canonical 链接
	reCitationPDF = regexp.MustCompile(`(?is)<meta[^>]*name=\"citation_pdf_url\"[^>]*content=\"([^\"]+)\"`)               // meta pdf 链接
	reDeliveryPDF = regexp.MustCompile(`(?is)<a[^>]*href=\"([^\"]*Delivery\\.cfm[^\"]+)\"`)                               // 备选 pdf 链接
)

func ExtractIDsFromSearchHTML(html string) []string {
	ids := []string{}
	if html == "" {
		return ids
	}
	matches := reAbsLink.FindAllStringSubmatch(html, -1)
	seen := map[string]struct{}{}
	for _, m := range matches {
		if len(m) < 2 {
			continue
		}
		id := m[1]
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

// 解析详情页标题与摘要（最小可用）
func ParseDetailTitleAbstract(html string) (title string, abstract string) {
	if html == "" {
		return "", ""
	}

	if m := reTitle.FindStringSubmatch(html); len(m) > 1 {
		title = strings.TrimSpace(m[1])
		// 去掉类似 " :: SSRN" 的尾缀
		title = strings.TrimSuffix(title, " :: SSRN")
	}
	if m := reAbsBox.FindStringSubmatch(html); len(m) > 1 {
		t := m[1]
		t = reTags.ReplaceAllString(t, " ")
		t = strings.Join(strings.Fields(t), " ")
		// 通常会以 "Abstract " 开头，去掉该前缀
		abstract = strings.TrimSpace(strings.TrimPrefix(t, "Abstract "))
	}
	return
}

// ParseDetailLinks 解析 canonical 页面 URL 与 PDF 下载链接
func ParseDetailLinks(html string) (canonical string, pdf string) {
	if html == "" {
		return "", ""
	}
	if m := reCanonical.FindStringSubmatch(html); len(m) > 1 {
		canonical = strings.TrimSpace(m[1])
	}
	if m := reCitationPDF.FindStringSubmatch(html); len(m) > 1 {
		pdf = strings.TrimSpace(m[1])
	} else if m := reDeliveryPDF.FindStringSubmatch(html); len(m) > 1 {
		pdf = strings.TrimSpace(m[1])
		if strings.HasPrefix(pdf, "/") {
			pdf = "https://papers.ssrn.com" + pdf
		} else if strings.HasPrefix(pdf, "Delivery.cfm") {
			pdf = "https://papers.ssrn.com/sol3/" + pdf
		}
	}
	return
}
