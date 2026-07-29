//go:build linux

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

const watchMask = unix.IN_CREATE |
	unix.IN_CLOSE_WRITE |
	unix.IN_ATTRIB |
	unix.IN_MOVED_FROM |
	unix.IN_MOVED_TO |
	unix.IN_DELETE |
	unix.IN_DELETE_SELF |
	unix.IN_MOVE_SELF |
	unix.IN_UNMOUNT |
	unix.IN_IGNORED

type event struct {
	WatchDescriptor int32  `json:"watchDescriptor"`
	Mask            uint32 `json:"mask"`
	Cookie          uint32 `json:"cookie"`
	Name            string `json:"name"`
}

type measurement struct {
	DirectoryWatches int           `json:"directoryWatches"`
	WatchDuration    time.Duration `json:"watchDuration"`
	RSSBeforeKiB     int64         `json:"rssBeforeKiB"`
	RSSAfterKiB      int64         `json:"rssAfterKiB"`
	FDsBefore        int           `json:"fdsBefore"`
	FDsAfter         int           `json:"fdsAfter"`
}

type report struct {
	Kernel                string      `json:"kernel"`
	MaxUserWatches        string      `json:"maxUserWatches"`
	MaxQueuedEvents       string      `json:"maxQueuedEvents"`
	EventNames            []string    `json:"eventNames"`
	RenameCookieMatched   bool        `json:"renameCookieMatched"`
	SymlinkReopenRejected bool        `json:"symlinkReopenRejected"`
	SymlinkReopenError    string      `json:"symlinkReopenError"`
	RootReplacementSeen   bool        `json:"rootReplacementSeen"`
	OverflowSeen          bool        `json:"overflowSeen"`
	OverflowEventsCreated int         `json:"overflowEventsCreated"`
	Scale                 measurement `json:"scale"`
	RuntimeLimitations    []string    `json:"runtimeLimitations"`
}

func main() {
	var watchDirectories int
	var overflowEvents int
	flag.IntVar(&watchDirectories, "watch-directories", 10_000, "number of directory watches for the scale probe")
	flag.IntVar(&overflowEvents, "overflow-events", 50_000, "events to create without draining the queue")
	flag.Parse()

	result, err := run(watchDirectories, overflowEvents)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(watchDirectories, overflowEvents int) (report, error) {
	if watchDirectories < 1 || overflowEvents < 1 {
		return report{}, errors.New("probe counts must be positive")
	}
	root, err := os.MkdirTemp("", "foliopath-wch-")
	if err != nil {
		return report{}, fmt.Errorf("create root: %w", err)
	}
	defer os.RemoveAll(root)

	fd, err := unix.InotifyInit1(unix.IN_CLOEXEC | unix.IN_NONBLOCK)
	if err != nil {
		return report{}, fmt.Errorf("inotify init: %w", err)
	}
	defer unix.Close(fd)

	_, err = unix.InotifyAddWatch(fd, root, watchMask)
	if err != nil {
		return report{}, fmt.Errorf("watch root: %w", err)
	}
	rootFD, err := unix.Open(root, unix.O_PATH|unix.O_DIRECTORY|unix.O_CLOEXEC, 0)
	if err != nil {
		return report{}, fmt.Errorf("open root anchor: %w", err)
	}
	defer unix.Close(rootFD)

	child := filepath.Join(root, "incoming")
	if err := os.Mkdir(child, 0o755); err != nil {
		return report{}, fmt.Errorf("create child: %w", err)
	}
	time.Sleep(20 * time.Millisecond)
	initialEvents, err := drain(fd, 500*time.Millisecond)
	if err != nil {
		return report{}, err
	}
	_, err = unix.InotifyAddWatch(fd, child, watchMask)
	if err != nil {
		return report{}, fmt.Errorf("watch new child: %w", err)
	}

	source := filepath.Join(child, "slow.jpg")
	file, err := os.Create(source)
	if err != nil {
		return report{}, fmt.Errorf("create media: %w", err)
	}
	if _, err := file.Write(bytes.Repeat([]byte{0x7f}, 4096)); err != nil {
		return report{}, fmt.Errorf("write media: %w", err)
	}
	if err := file.Sync(); err != nil {
		return report{}, fmt.Errorf("sync media: %w", err)
	}
	if err := file.Close(); err != nil {
		return report{}, fmt.Errorf("close media: %w", err)
	}
	target := filepath.Join(child, "ready.jpg")
	if err := os.Rename(source, target); err != nil {
		return report{}, fmt.Errorf("rename media: %w", err)
	}
	if err := os.Remove(target); err != nil {
		return report{}, fmt.Errorf("delete media: %w", err)
	}
	mediaEvents, err := drain(fd, 750*time.Millisecond)
	if err != nil {
		return report{}, err
	}

	outside, err := os.CreateTemp("", "foliopath-wch-outside-")
	if err != nil {
		return report{}, fmt.Errorf("create outside file: %w", err)
	}
	outsidePath := outside.Name()
	_ = outside.Close()
	defer os.Remove(outsidePath)
	linkPath := filepath.Join(root, "escape.jpg")
	if err := os.Symlink(outsidePath, linkPath); err != nil {
		return report{}, fmt.Errorf("create symlink: %w", err)
	}
	symlinkRejected, symlinkError := safeOpenRejected(rootFD, "escape.jpg")

	movedRoot := root + "-old"
	if err := os.Rename(root, movedRoot); err != nil {
		return report{}, fmt.Errorf("move watched root: %w", err)
	}
	defer os.RemoveAll(movedRoot)
	if err := os.Mkdir(root, 0o755); err != nil {
		return report{}, fmt.Errorf("replace watched root: %w", err)
	}
	replacementSeen, err := rootIdentityChanged(rootFD, root)
	if err != nil {
		return report{}, err
	}
	replacementEvents, err := drain(fd, 500*time.Millisecond)
	if err != nil {
		return report{}, err
	}

	scaleRoot := filepath.Join(root, "scale")
	if err := os.Mkdir(scaleRoot, 0o755); err != nil {
		return report{}, fmt.Errorf("create scale root: %w", err)
	}
	rssBefore := rssKiB()
	fdsBefore := fdCount()
	started := time.Now()
	for index := 0; index < watchDirectories; index++ {
		directory := filepath.Join(scaleRoot, fmt.Sprintf("d%05d", index))
		if err := os.Mkdir(directory, 0o755); err != nil {
			return report{}, fmt.Errorf("create scale directory %d: %w", index, err)
		}
		if _, err := unix.InotifyAddWatch(fd, directory, watchMask); err != nil {
			return report{}, fmt.Errorf("watch scale directory %d: %w", index, err)
		}
	}
	scale := measurement{
		DirectoryWatches: watchDirectories + 2,
		WatchDuration:    time.Since(started),
		RSSBeforeKiB:     rssBefore,
		RSSAfterKiB:      rssKiB(),
		FDsBefore:        fdsBefore,
		FDsAfter:         fdCount(),
	}
	_, _ = drain(fd, 250*time.Millisecond)

	overflowRoot := filepath.Join(root, "overflow")
	if err := os.Mkdir(overflowRoot, 0o755); err != nil {
		return report{}, fmt.Errorf("create overflow root: %w", err)
	}
	if _, err := unix.InotifyAddWatch(fd, overflowRoot, watchMask); err != nil {
		return report{}, fmt.Errorf("watch overflow root: %w", err)
	}
	for index := 0; index < overflowEvents; index++ {
		path := filepath.Join(overflowRoot, fmt.Sprintf("e%06d", index))
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o644)
		if err != nil {
			return report{}, fmt.Errorf("create overflow event %d: %w", index, err)
		}
		if err := file.Close(); err != nil {
			return report{}, fmt.Errorf("close overflow event %d: %w", index, err)
		}
	}
	overflowBatch, err := drain(fd, 2*time.Second)
	if err != nil {
		return report{}, err
	}

	allEvents := append(initialEvents, mediaEvents...)
	allEvents = append(allEvents, replacementEvents...)
	allEvents = append(allEvents, overflowBatch...)
	return report{
		Kernel:                strings.TrimSpace(readFile("/proc/sys/kernel/osrelease")),
		MaxUserWatches:        strings.TrimSpace(readFile("/proc/sys/fs/inotify/max_user_watches")),
		MaxQueuedEvents:       strings.TrimSpace(readFile("/proc/sys/fs/inotify/max_queued_events")),
		EventNames:            summarize(allEvents),
		RenameCookieMatched:   renameCookieMatched(mediaEvents),
		SymlinkReopenRejected: symlinkRejected,
		SymlinkReopenError:    symlinkError,
		RootReplacementSeen:   replacementSeen || hasMask(replacementEvents, unix.IN_MOVE_SELF),
		OverflowSeen:          hasMask(overflowBatch, unix.IN_Q_OVERFLOW),
		OverflowEventsCreated: overflowEvents,
		Scale:                 scale,
		RuntimeLimitations: []string{
			"ENOSPC depends on host sysctl and is reported, not modified by this probe",
			"mount/unmount injection requires a separate privileged mount-namespace test",
			"results from emulated Linux do not replace native amd64/arm64 evidence",
		},
	}, nil
}

func drain(fd int, quiet time.Duration) ([]event, error) {
	deadline := time.Now().Add(quiet)
	buffer := make([]byte, 1<<20)
	var events []event
	for time.Now().Before(deadline) {
		count, err := unix.Read(fd, buffer)
		if err != nil {
			if errors.Is(err, unix.EAGAIN) {
				time.Sleep(5 * time.Millisecond)
				continue
			}
			return nil, fmt.Errorf("read inotify: %w", err)
		}
		offset := 0
		for offset+unix.SizeofInotifyEvent <= count {
			raw := (*unix.InotifyEvent)(unsafe.Pointer(&buffer[offset]))
			nameBytes := buffer[offset+unix.SizeofInotifyEvent : offset+unix.SizeofInotifyEvent+int(raw.Len)]
			name := strings.TrimRight(string(nameBytes), "\x00")
			events = append(events, event{
				WatchDescriptor: raw.Wd,
				Mask:            raw.Mask,
				Cookie:          raw.Cookie,
				Name:            name,
			})
			offset += unix.SizeofInotifyEvent + int(raw.Len)
		}
		deadline = time.Now().Add(20 * time.Millisecond)
	}
	return events, nil
}

func safeOpenRejected(rootFD int, relativePath string) (bool, string) {
	fd, err := unix.Openat2(rootFD, relativePath, &unix.OpenHow{
		Flags:   unix.O_RDONLY | unix.O_CLOEXEC,
		Resolve: unix.RESOLVE_BENEATH | unix.RESOLVE_NO_SYMLINKS | unix.RESOLVE_NO_XDEV,
	})
	if err == nil {
		_ = unix.Close(fd)
		return false, ""
	}
	return true, err.Error()
}

func rootIdentityChanged(rootFD int, configuredPath string) (bool, error) {
	var anchored unix.Stat_t
	if err := unix.Fstat(rootFD, &anchored); err != nil {
		return false, fmt.Errorf("stat anchored root: %w", err)
	}
	var configured unix.Stat_t
	if err := unix.Stat(configuredPath, &configured); err != nil {
		return false, fmt.Errorf("stat configured root: %w", err)
	}
	return anchored.Dev != configured.Dev || anchored.Ino != configured.Ino, nil
}

func renameCookieMatched(events []event) bool {
	from := map[uint32]struct{}{}
	for _, item := range events {
		if item.Mask&unix.IN_MOVED_FROM != 0 {
			from[item.Cookie] = struct{}{}
		}
	}
	for _, item := range events {
		if item.Mask&unix.IN_MOVED_TO != 0 {
			if _, ok := from[item.Cookie]; ok && item.Cookie != 0 {
				return true
			}
		}
	}
	return false
}

func hasMask(events []event, mask uint32) bool {
	for _, item := range events {
		if item.Mask&mask != 0 {
			return true
		}
	}
	return false
}

func summarize(events []event) []string {
	names := map[string]struct{}{}
	for _, item := range events {
		for mask, name := range map[uint32]string{
			unix.IN_CREATE:      "create",
			unix.IN_CLOSE_WRITE: "close_write",
			unix.IN_ATTRIB:      "attrib",
			unix.IN_MOVED_FROM:  "moved_from",
			unix.IN_MOVED_TO:    "moved_to",
			unix.IN_DELETE:      "delete",
			unix.IN_DELETE_SELF: "delete_self",
			unix.IN_MOVE_SELF:   "move_self",
			unix.IN_UNMOUNT:     "unmount",
			unix.IN_IGNORED:     "ignored",
			unix.IN_Q_OVERFLOW:  "queue_overflow",
		} {
			if item.Mask&mask != 0 {
				names[name] = struct{}{}
			}
		}
	}
	result := make([]string, 0, len(names))
	for _, name := range []string{
		"create", "close_write", "attrib", "moved_from", "moved_to", "delete",
		"delete_self", "move_self", "unmount", "ignored", "queue_overflow",
	} {
		if _, ok := names[name]; ok {
			result = append(result, name)
		}
	}
	return result
}

func rssKiB() int64 {
	for _, line := range strings.Split(readFile("/proc/self/status"), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 3 && fields[0] == "VmRSS:" {
			value, _ := strconv.ParseInt(fields[1], 10, 64)
			return value
		}
	}
	return -1
}

func fdCount() int {
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		return -1
	}
	return len(entries)
}

func readFile(path string) string {
	value, _ := os.ReadFile(path)
	return string(value)
}
