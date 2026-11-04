package public

import "strings"

// SplitRules 按逗号或换行拆分后台配置的规则列表，忽略空项
func SplitRules(s string) []string {
	out := []string{}
	for _, item := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '\n' || r == '\r' }) {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}

// InList 判断 item 是否在列表中
func InList(item string, list []string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}
