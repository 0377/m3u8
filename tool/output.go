package tool

import (
	"fmt"
	"path/filepath"
	"strings"
)

const defaultOutputBaseName = "main"

// ResolveOutputBaseName 从用户指定的输出文件名解析不含扩展名的基名。
// 支持 "video"、"video.ts"、"video.mp4" 等形式；空字符串返回默认 "main"。
func ResolveOutputBaseName(filename string) (string, error) {
	if filename == "" {
		return defaultOutputBaseName, nil
	}

	if strings.ContainsAny(filename, `/\`) || filename != filepath.Base(filename) {
		return "", fmt.Errorf("输出文件名不能包含路径: %s", filename)
	}

	name := filename

	ext := strings.ToLower(filepath.Ext(name))
	base := strings.TrimSuffix(name, ext)
	if base == "" {
		return "", fmt.Errorf("无效的输出文件名: %s", filename)
	}
	return base, nil
}
