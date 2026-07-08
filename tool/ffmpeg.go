package tool

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ConvertTSToMP4 使用 ffmpeg 将 TS 流复制封装为 MP4（不重新编码）。
func ConvertTSToMP4(input, output string) error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("未找到 ffmpeg，请先安装: brew install ffmpeg")
	}
	if _, err := os.Stat(input); err != nil {
		return fmt.Errorf("输入文件不存在: %s", input)
	}
	if err := os.MkdirAll(filepath.Dir(output), os.ModePerm); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	cmd := exec.Command("ffmpeg",
		"-hide_banner", "-loglevel", "error", "-stats",
		"-i", input,
		"-c", "copy",
		"-bsf:a", "aac_adtstoasc",
		"-movflags", "+faststart",
		"-y", output,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("[convert] %s -> %s\n", input, output)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg 转换失败: %w", err)
	}
	fmt.Printf("[output] %s\n", output)
	return nil
}
