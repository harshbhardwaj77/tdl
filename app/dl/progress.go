package dl

import (
	"context"
	stdErrors "errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fatih/color"
	"github.com/gabriel-vasile/mimetype"
	"github.com/go-faster/errors"
	pw "github.com/jedib0t/go-pretty/v6/progress"

	"github.com/iyear/tdl/core/downloader"
	"github.com/iyear/tdl/core/util/fsutil"
	"github.com/iyear/tdl/pkg/prog"
	"github.com/iyear/tdl/pkg/utils"
)

type progress struct {
	pw       pw.Writer
	trackers *sync.Map // map[ID]*pw.Tracker
	opts     Options

	it *iter
}

func newProgress(p pw.Writer, it *iter, opts Options) *progress {
	return &progress{
		pw:       p,
		trackers: &sync.Map{},
		opts:     opts,
		it:       it,
	}
}

func (p *progress) OnAdd(elem downloader.Elem) {
	tracker := prog.AppendTracker(p.pw, utils.Byte.FormatBinaryBytes, p.processMessage(elem), elem.File().Size())
	p.trackers.Store(elem.(*iterElem).id, tracker)
}

func (p *progress) OnDownload(elem downloader.Elem, state downloader.ProgressState) {
	tracker, ok := p.trackers.Load(elem.(*iterElem).id)
	if !ok {
		return
	}

	t := tracker.(*pw.Tracker)
	t.UpdateTotal(state.Total)
	t.SetValue(state.Downloaded)
}

func (p *progress) OnDone(elem downloader.Elem, err error) {
	e := elem.(*iterElem)

	tracker, ok := p.trackers.Load(e.id)
	if !ok {
		return
	}
	t := tracker.(*pw.Tracker)

	// Optional: ensure any buffered data is flushed to disk before closing/renaming.
	// Ignore error here; Close() will surface issues too.
	_ = e.to.Sync()

	if err := e.to.Close(); err != nil {
		p.fail(t, elem, errors.Wrap(err, "close file"))
		return
	}

	if err != nil {
		if !errors.Is(err, context.Canceled) { // don't report user cancel
			p.fail(t, elem, errors.Wrap(err, "progress"))
		}
		_ = os.Remove(e.to.Name()) // just try to remove temp file, ignore error
		return
	}

	p.it.Finish(e.logicalPos)

	if err := p.donePost(e); err != nil {
		p.fail(t, elem, errors.Wrap(err, "post file"))
		return
	}
}

func (p *progress) donePost(elem *iterElem) error {
	newfile := strings.TrimSuffix(filepath.Base(elem.to.Name()), tempExt)

	if p.opts.RewriteExt {
		mime, err := mimetype.DetectFile(elem.to.Name())
		if err != nil {
			return errors.Wrap(err, "detect mime")
		}
		ext := mime.Extension()
		if ext != "" && (filepath.Ext(newfile) != ext) {
			newfile = fsutil.GetNameWithoutExt(newfile) + ext
		}
	}

	newpath := filepath.Join(filepath.Dir(elem.to.Name()), newfile)

	// Windows can temporarily lock files (Defender/AV/Indexer/Explorer preview).
	// Retry rename to avoid failing the download at the final step.
	if err := renameWithRetry(elem.to.Name(), newpath); err != nil {
		return errors.Wrap(err, "rename file")
	}

	// Set file modification time to message date if available
	if elem.file.Date > 0 {
		fileTime := time.Unix(elem.file.Date, 0)
		if err := os.Chtimes(newpath, fileTime, fileTime); err != nil {
			return errors.Wrap(err, "set file time")
		}
	}

	return nil
}

func (p *progress) fail(t *pw.Tracker, elem downloader.Elem, err error) {
	p.pw.Log(color.RedString("%s error: %s", p.elemString(elem), err.Error()))
	t.MarkAsErrored()
}

func (p *progress) processMessage(elem downloader.Elem) string {
	return p.elemString(elem)
}

func (p *progress) elemString(elem downloader.Elem) string {
	e := elem.(*iterElem)
	return fmt.Sprintf("%s(%d):%d -> %s",
		e.from.VisibleName(),
		e.from.ID(),
		e.fromMsg.ID,
		strings.TrimSuffix(e.to.Name(), tempExt))
}

func renameWithRetry(oldpath, newpath string) error {
	const (
		// On some Windows machines (heavy AV/Defender, slow disks, etc.),
		// the temp file or destination can stay locked for quite a while
		// after we close our own handle. A small retry window (~9s) is
		// often not enough for large media files, which leads to
		// "post file: rename file" errors and forces a re-download.
		//
		// We therefore allow a much longer retry window here on Windows
		// (attempts * delay), while still bailing out quickly on other
		// platforms or non-lock related errors.
		attempts = 2000
		delay    = 100 * time.Millisecond
	)

	var err error
	for i := 0; i < attempts; i++ {
		err = os.Rename(oldpath, newpath)
		if err == nil {
			return nil
		}

		// Only retry transient Windows locking errors.
		if runtime.GOOS != "windows" || !isWindowsFileLockError(err) {
			return err
		}

		time.Sleep(delay)
	}
	return err
}

func isWindowsFileLockError(err error) bool {
	// Numeric errno values so this compiles cross-platform.
	// 5  = Access is denied
	// 32 = Sharing violation
	// 33 = Lock violation
	const (
		winAccessDenied     syscall.Errno = 5
		winSharingViolation syscall.Errno = 32
		winLockViolation    syscall.Errno = 33
	)

	for err != nil {
		if pe, ok := err.(*os.PathError); ok {
			err = pe.Err
			continue
		}

		if errno, ok := err.(syscall.Errno); ok {
			return errno == winAccessDenied ||
				errno == winSharingViolation ||
				errno == winLockViolation
		}

		err = stdErrors.Unwrap(err)
	}
	return false
}
