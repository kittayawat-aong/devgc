package scan

import (
	"io/fs"
	"path/filepath"
	"sync"
	"syscall"
)

// Size fills in SizeKB for every target, measuring several at once.
// It counts disk blocks like `du -sk`.
func Size(targets []*Target, workers int) {
	jobs := make(chan *Target)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for t := range jobs {
				t.SizeKB = diskKB(t.Path)
			}
		}()
	}
	for _, t := range targets {
		jobs <- t
	}
	close(jobs)
	wg.Wait()
}

func diskKB(root string) int64 {
	var blocks int64
	_ = filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // unreadable: skip
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		if st, ok := info.Sys().(*syscall.Stat_t); ok {
			blocks += st.Blocks // 512-byte blocks
		}
		return nil
	})
	return blocks / 2
}
