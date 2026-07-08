package dl

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/0377/m3u8/crypt"
	"github.com/0377/m3u8/parse"
	"github.com/0377/m3u8/tool"
)

const (
	tsExt            = ".ts"
	tsFolderName     = "ts"
	defaultBaseName  = "main"
	tsTempFileSuffix = "_tmp"
	progressWidth    = 40
)

type ProgressReporter func(done, total int, message string)

type Downloader struct {
	lock      sync.Mutex
	queue     []int
	folder    string
	tsFolder  string
	outputTS  string
	outputMP4 string
	retries   map[int]int
	failed    map[int]struct{}
	maxRetry  int
	finish    int32
	segLen    int
	reporter  ProgressReporter
	cancelCtx context.Context

	result   *parse.Result
	httpCfg  *tool.HTTPConfig
	cryptSvc *crypt.Service
}

func (d *Downloader) SetProgressReporter(fn ProgressReporter) {
	d.reporter = fn
}

// SetCancelContext sets a context for cooperative cancellation during download.
func (d *Downloader) SetCancelContext(ctx context.Context) {
	d.cancelCtx = ctx
}

func (d *Downloader) cancelled() bool {
	return d.cancelCtx != nil && d.cancelCtx.Err() != nil
}

func (d *Downloader) reportProgress(message string) {
	done := int(atomic.LoadInt32(&d.finish))
	if d.reporter != nil {
		d.reporter(done, d.segLen, message)
		return
	}
	if message == "downloading" && done > 0 {
		fmt.Printf("[download %6.2f%%]\n", float32(done)/float32(d.segLen)*100)
	}
}

// NewTask returns a Task instance.
// filename is the output base name or filename (e.g. "video", "video.mp4"); empty uses "main".
func NewTask(output string, url string, filename string, httpCfg *tool.HTTPConfig, cryptSvc *crypt.Service) (*Downloader, error) {
	result, err := parse.FromURL(url, httpCfg, cryptSvc)
	if err != nil {
		return nil, err
	}
	var folder string
	if output == "" {
		output = "."
	}
	folder, err = filepath.Abs(output)
	if err != nil {
		return nil, fmt.Errorf("resolve output folder failed: %s", err.Error())
	}
	if err := os.MkdirAll(folder, os.ModePerm); err != nil {
		return nil, fmt.Errorf("create storage folder failed: %s", err.Error())
	}
	tsPath, mp4Path, err := resolveOutputPaths(folder, filename)
	if err != nil {
		return nil, err
	}
	tsFolder := filepath.Join(folder, tsFolderName)
	if err := os.MkdirAll(tsFolder, os.ModePerm); err != nil {
		return nil, fmt.Errorf("create ts folder '[%s]' failed: %s", tsFolder, err.Error())
	}
	segLen := len(result.M3u8.Segments)
	currentMeta := NewTaskMeta(url, filename, segLen, time.Now().Format(time.RFC3339))

	existingMeta, err := LoadTaskMeta(folder)
	if err != nil {
		return nil, err
	}
	if existingMeta == nil {
		if err := SaveTaskMeta(folder, currentMeta); err != nil {
			return nil, err
		}
	} else {
		if err := ValidateTaskMeta(existingMeta, currentMeta); err != nil {
			return nil, err
		}
	}

	completed, err := scanCompletedSegments(tsFolder, segLen)
	if err != nil {
		return nil, err
	}

	if existingMeta == nil && len(completed) > 0 {
		fmt.Printf("[info] 未发现任务元数据，创建新任务（ts/ 中已有 %d 个分片将被复用）\n", len(completed))
	}
	if existingMeta != nil && len(completed) > 0 {
		fmt.Printf("[resume] 已完成 %d/%d 分片，继续下载剩余 %d 个\n", len(completed), segLen, segLen-len(completed))
	}

	d := &Downloader{
		folder:    folder,
		tsFolder:  tsFolder,
		outputTS:  tsPath,
		outputMP4: mp4Path,
		retries:   make(map[int]int),
		failed:    make(map[int]struct{}),
		result:   result,
		httpCfg:  httpCfg,
		cryptSvc: cryptSvc,
	}
	d.segLen = segLen
	d.queue = buildQueue(segLen, completed)
	atomic.StoreInt32(&d.finish, int32(len(completed)))
	return d, nil
}

// Start runs downloader. When toMP4 is true, merged TS is converted to MP4 via ffmpeg.
func (d *Downloader) Start(concurrency int, toMP4 bool, maxRetry int) error {
	if d.cryptSvc != nil {
		defer func() { _ = d.cryptSvc.Close() }()
	}
	if d.cancelled() {
		return d.cancelCtx.Err()
	}
	d.maxRetry = maxRetry
	var wg sync.WaitGroup
	// struct{} zero size
	limitChan := make(chan struct{}, concurrency)
	for {
		if d.cancelled() {
			wg.Wait()
			return d.cancelCtx.Err()
		}
		tsIdx, end, err := d.next()
		if err != nil {
			if end {
				break
			}
			continue
		}
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if err := d.download(idx); err != nil {
				fmt.Printf("[failed] %s\n", err.Error())
				if backErr := d.back(idx); backErr != nil {
					fmt.Printf("[give up] segment %d: %s\n", idx, backErr.Error())
				}
			}
			<-limitChan
		}(tsIdx)
		limitChan <- struct{}{}
	}
	wg.Wait()
	if d.cancelled() {
		return d.cancelCtx.Err()
	}
	if len(d.failed) > 0 {
		return fmt.Errorf("%d segments failed after %d retries", len(d.failed), maxRetry)
	}
	if err := d.merge(); err != nil {
		return err
	}
	if d.cancelled() {
		return d.cancelCtx.Err()
	}
	if toMP4 {
		if err := tool.ConvertTSToMP4(d.outputTS, d.outputMP4); err != nil {
			return err
		}
		if err := os.Remove(d.outputTS); err != nil {
			return fmt.Errorf("删除中间 TS 文件失败: %w", err)
		}
		fmt.Printf("[cleanup] 已删除中间文件 %s\n", d.outputTS)
	}
	return nil
}

func (d *Downloader) download(segIndex int) error {
	if d.cancelled() {
		return d.cancelCtx.Err()
	}
	tsFilename := tsFilename(segIndex)
	tsUrl := d.tsURL(segIndex)
	b, e := tool.Get(tsUrl, d.httpCfg)
	if e != nil {
		return fmt.Errorf("request %s, %s", tsUrl, e.Error())
	}
	//noinspection GoUnhandledErrorResult
	defer b.Close()
	fPath := filepath.Join(d.tsFolder, tsFilename)
	fTemp := fPath + tsTempFileSuffix
	f, err := os.Create(fTemp)
	if err != nil {
		return fmt.Errorf("create file: %s, %s", tsFilename, err.Error())
	}
	bytes, err := io.ReadAll(b)
	if err != nil {
		return fmt.Errorf("read bytes: %s, %s", tsUrl, err.Error())
	}
	sf := d.result.M3u8.Segments[segIndex]
	if sf == nil {
		return fmt.Errorf("invalid segment index: %d", segIndex)
	}
	keyMat, ok := d.result.Keys[sf.KeyIndex]
	if ok && len(keyMat.Key) > 0 {
		iv := keyMat.IV
		if len(iv) == 0 && d.result.M3u8.Keys[sf.KeyIndex] != nil {
			iv = []byte(d.result.M3u8.Keys[sf.KeyIndex].IV)
		}
		ctx := &crypt.Context{
			M3U8URL:    d.result.URL.String(),
			SegmentURI: sf.URI,
			SegmentIdx: segIndex,
			Key:        keyMat.Key,
			IV:         iv,
		}
		if k := d.result.M3u8.Keys[sf.KeyIndex]; k != nil {
			ctx.Method = string(k.Method)
			ctx.KeyMeta = crypt.KeyMeta{
				Method: string(k.Method),
				URI:    k.URI,
				IV:     k.IV,
			}
		}
		if d.cryptSvc != nil {
			bytes, err = d.cryptSvc.DecryptSegment(ctx, bytes, keyMat.Key, iv)
		} else {
			bytes, err = tool.AES128Decrypt(bytes, keyMat.Key, iv)
		}
		if err != nil {
			return fmt.Errorf("decrypt: %s, %s", tsUrl, err.Error())
		}
	}
	// https://en.wikipedia.org/wiki/MPEG_transport_stream
	// Some TS files do not start with SyncByte 0x47, they can not be played after merging,
	// Need to remove the bytes before the SyncByte 0x47(71).
	syncByte := uint8(71) //0x47
	bLen := len(bytes)
	for j := 0; j < bLen; j++ {
		if bytes[j] == syncByte {
			bytes = bytes[j:]
			break
		}
	}
	w := bufio.NewWriter(f)
	if _, err := w.Write(bytes); err != nil {
		return fmt.Errorf("write to %s: %s", fTemp, err.Error())
	}
	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush %s: %s", fTemp, err.Error())
	}
	// Release file resource to rename file
	_ = f.Close()
	if err = os.Rename(fTemp, fPath); err != nil {
		return err
	}
	// Maybe it will be safer in this way...
	atomic.AddInt32(&d.finish, 1)
	d.reportProgress("downloading")
	return nil
}

func (d *Downloader) next() (segIndex int, end bool, err error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	if len(d.queue) == 0 {
		err = fmt.Errorf("queue empty")
		finish := atomic.LoadInt32(&d.finish)
		if finish == int32(d.segLen) {
			end = true
			return
		}
		if int(finish)+len(d.failed) == d.segLen {
			end = true
			return
		}
		// Some segment indexes are still running.
		end = false
		return
	}
	segIndex = d.queue[0]
	d.queue = d.queue[1:]
	return
}

func (d *Downloader) back(segIndex int) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	if sf := d.result.M3u8.Segments[segIndex]; sf == nil {
		return fmt.Errorf("invalid segment index: %d", segIndex)
	}
	if _, givenUp := d.failed[segIndex]; givenUp {
		return fmt.Errorf("segment %d already given up", segIndex)
	}

	d.retries[segIndex]++
	if d.retries[segIndex] > d.maxRetry {
		d.failed[segIndex] = struct{}{}
		return fmt.Errorf("exceeded max retries (%d)", d.maxRetry)
	}

	d.reportProgress("retrying")
	d.queue = append(d.queue, segIndex)
	return nil
}

func (d *Downloader) merge() error {
	if d.cancelled() {
		return d.cancelCtx.Err()
	}
	// In fact, the number of downloaded segments should be equal to number of m3u8 segments
	missingCount := 0
	for idx := 0; idx < d.segLen; idx++ {
		tsFilename := tsFilename(idx)
		f := filepath.Join(d.tsFolder, tsFilename)
		if _, err := os.Stat(f); err != nil {
			missingCount++
		}
	}
	if missingCount > 0 {
		fmt.Printf("[warning] %d files missing\n", missingCount)
	}

	// Create a TS file for merging, all segment files will be written to this file.
	mFilePath := d.outputTS
	mFile, err := os.Create(mFilePath)
	if err != nil {
		return fmt.Errorf("create main TS file failed：%s", err.Error())
	}
	//noinspection GoUnhandledErrorResult
	defer mFile.Close()

	writer := bufio.NewWriter(mFile)
	mergedCount := 0
	for segIndex := 0; segIndex < d.segLen; segIndex++ {
		if d.cancelled() {
			return d.cancelCtx.Err()
		}
		tsFilename := tsFilename(segIndex)
		segFile, err := os.Open(filepath.Join(d.tsFolder, tsFilename))
		if err != nil {
			continue
		}
		_, err = io.Copy(writer, segFile)
		_ = segFile.Close()
		if err != nil {
			continue
		}
		mergedCount++
		if d.reporter != nil {
			d.reportProgress("merging")
		} else {
			tool.DrawProgressBar("merge",
				float32(mergedCount)/float32(d.segLen), progressWidth)
		}
	}
	_ = writer.Flush()

	if mergedCount != d.segLen {
		fmt.Printf("[warning] \n%d files merge failed", d.segLen-mergedCount)
		return fmt.Errorf("%d files merge failed", d.segLen-mergedCount)
	}

	// Remove `ts` folder and task meta only after successful merge
	_ = os.RemoveAll(d.tsFolder)
	if err := RemoveTaskMeta(d.folder); err != nil {
		return fmt.Errorf("remove task meta failed: %w", err)
	}

	fmt.Printf("\n[output] %s\n", mFilePath)

	return nil
}

func (d *Downloader) tsURL(segIndex int) string {
	seg := d.result.M3u8.Segments[segIndex]
	return tool.ResolveURL(d.result.URL, seg.URI)
}

func tsFilename(ts int) string {
	return strconv.Itoa(ts) + tsExt
}

func genSlice(len int) []int {
	s := make([]int, 0)
	for i := 0; i < len; i++ {
		s = append(s, i)
	}
	return s
}

func resolveOutputPaths(dir, filename string) (tsPath, mp4Path string, err error) {
	baseName, err := tool.ResolveOutputBaseName(filename)
	if err != nil {
		return "", "", err
	}
	tsPath = filepath.Join(dir, baseName+tsExt)
	mp4Path = filepath.Join(dir, baseName+".mp4")
	return tsPath, mp4Path, nil
}
