package dl

import (
	"fmt"
	"os"
	"path/filepath"
)

const tsSyncByte = 0x47

func isValidSegmentFile(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if info.Size() == 0 {
		return false, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	buf := make([]byte, 1)
	n, err := f.Read(buf)
	if err != nil {
		return false, err
	}
	if n == 0 {
		return false, nil
	}
	return buf[0] == tsSyncByte, nil
}

func removeSegmentFiles(fPath, tmpPath string) {
	_ = os.Remove(fPath)
	_ = os.Remove(tmpPath)
}

func scanCompletedSegments(tsFolder string, segLen int) (map[int]struct{}, error) {
	completed := make(map[int]struct{})
	for idx := 0; idx < segLen; idx++ {
		fPath := filepath.Join(tsFolder, tsFilename(idx))
		tmpPath := fPath + tsTempFileSuffix

		if _, err := os.Stat(tmpPath); err == nil {
			_ = os.Remove(tmpPath)
		}

		if _, err := os.Stat(fPath); err == nil {
			valid, err := isValidSegmentFile(fPath)
			if err != nil {
				return nil, fmt.Errorf("validate segment %d: %w", idx, err)
			}
			if !valid {
				removeSegmentFiles(fPath, tmpPath)
				continue
			}
			completed[idx] = struct{}{}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("stat segment %d: %w", idx, err)
		}
	}
	return completed, nil
}

func wipeTSFolder(tsFolder string) error {
	entries, err := os.ReadDir(tsFolder)
	if err != nil {
		return fmt.Errorf("read ts folder: %w", err)
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(tsFolder, e.Name())); err != nil {
			return fmt.Errorf("remove ts file %s: %w", e.Name(), err)
		}
	}
	return nil
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
