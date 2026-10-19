//go:build development

package desktop

// DevelopmentOrigin is set only in a separately tagged development build.
var DevelopmentOrigin = "http://127.0.0.1:3000"

func BuildOrigin() (string, bool) { return DevelopmentOrigin, true }
