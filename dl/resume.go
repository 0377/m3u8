package dl

import (
	"fmt"
	"os"
	"path/filepath"
)

func scanCompletedSegments(tsFolder string, segLen int) (map[int]struct{}, error) {
	completed := make(map[int]struct{})
	for idx := 0; idx < segLen; idx++ {
		fPath := filepath.Join(tsFolder, tsFilename(idx))
		tmpPath := fPath + tsTempFileSuffix

		if _, err := os.Stat(tmpPath); err == nil {
			_ = os.Remove(tmpPath)
		}

		if _, err := os.Stat(fPath); err == nil {
			completed[idx] = struct{}{}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat segment %d: %w", idx, err)
		}
	}
	return completed, nil
}

func buildQueue(segLen int, completed map[int]struct{}) []int {
	queue := make([]int, 0, segLen-len(completed))
	for i := 0; i < segLen; i++ {
		if _, ok := completed[i]; !ok {
			queue = append(queue, i)
		}
	}
	return queue
}
