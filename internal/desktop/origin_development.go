//go:build development

package desktop

func BuildOrigin() (string, bool) { return ServerOrigin, true }
