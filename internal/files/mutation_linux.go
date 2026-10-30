//go:build linux

package files

import (
	"bufio"
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/HappyQuQu/foliopath/internal/deletion"
	"golang.org/x/sys/unix"
)

func cloneMutationRoot(root *Root) (*os.File, error) {
	root.mu.RLock()
	defer root.mu.RUnlock()
	if root.closed {
		return nil, deletion.ErrWriteUnavailable
	}
	if _, err := root.captureLocked(); err != nil {
		return nil, deletion.ErrWriteUnavailable
	}
	fd, err := unix.FcntlInt(root.anchor.boundary.Fd(), unix.F_DUPFD_CLOEXEC, 0)
	if err != nil {
		return nil, deletion.ErrWriteUnavailable
	}
	return os.NewFile(uintptr(fd), "media-root"), nil
}

type mutationState struct {
	root            *anchoredRoot
	removal         *mutationRemoval
	dirty           map[string]struct{}
	libraryRoot     string
	libraryIdentity deletion.SourceIdentity
}

func newMutationState(root *Root) (*mutationState, error) {
	file, err := cloneMutationRoot(root)
	if err != nil {
		return nil, err
	}
	state := &mutationState{root: &anchoredRoot{file: file, boundary: file}}
	if err := mutationRootSupported(int(file.Fd())); err != nil {
		_ = file.Close()
		return nil, err
	}
	if err := state.rejectNestedMounts(); err != nil {
		_ = file.Close()
		return nil, err
	}
	return state, nil
}

func (state *mutationState) close() error {
	state.closeRemoval()
	return state.root.Close()
}

func mutationRootSupported(fd int) error {
	var info unix.Statfs_t
	if unix.Fstatfs(fd, &info) != nil || info.Flags&unix.ST_RDONLY != 0 {
		return deletion.ErrWriteUnavailable
	}
	switch uint64(info.Type) {
	case unix.EXT4_SUPER_MAGIC, unix.XFS_SUPER_MAGIC, unix.BTRFS_SUPER_MAGIC, unix.TMPFS_MAGIC, 0x2fc12fc1:
		return nil
	default:
		return deletion.ErrWriteUnavailable
	}
}

func validateMutationRequest(request mutationRequest) error {
	switch request.Kind {
	case "probe":
		return nil
	case "inspect", "bind", "remove", "trash", "restore":
		normalized, err := Normalize(request.Path)
		if err != nil || normalized != request.Path {
			return deletion.ErrInvalid
		}
		if request.Kind != "inspect" && (normalized == deletion.TrashDirectory || strings.HasPrefix(normalized, deletion.TrashDirectory+"/")) {
			return deletion.ErrInvalid
		}
		if request.Kind == "inspect" {
			return nil
		}
		if request.Kind == "bind" {
			if !request.Expected.Directory || request.Expected.Inode == 0 {
				return deletion.ErrInvalid
			}
			return nil
		}
		if normalized == "" {
			return deletion.ErrInvalid
		}
		if request.Expected.Inode == 0 {
			return deletion.ErrInvalid
		}
		if request.Kind == "remove" {
			return nil
		}
	case "prepare", "purge", "inspect_trash":
	default:
		return deletion.ErrInvalid
	}
	if !mutationToken(request.Instance) {
		return deletion.ErrInvalid
	}
	if request.Kind != "prepare" && !mutationToken(request.Entry) {
		return deletion.ErrInvalid
	}
	if request.Kind == "purge" && request.Expected.Inode == 0 {
		return deletion.ErrInvalid
	}
	return nil
}

func mutationToken(value string) bool {
	if len(value) < 1 || len(value) > 64 {
		return false
	}
	for _, character := range value {
		if character >= 'a' && character <= 'z' || character >= '0' && character <= '9' || character == '-' || character == '_' {
			continue
		}
		return false
	}
	return true
}

func (state *mutationState) execute(ctx context.Context, request mutationRequest) (mutationReply, error) {
	if (request.Kind == "inspect" || request.Kind == "inspect_trash") && state.libraryIdentity.Inode != 0 {
		if err := state.verifyLibrary(); err != nil {
			return mutationReply{}, err
		}
	}
	if request.Kind == "probe" || request.Kind == "prepare" {
		if err := state.rejectNestedMounts(); err != nil {
			return mutationReply{}, err
		}
	}
	if request.Kind == "trash" || request.Kind == "restore" || request.Kind == "remove" || request.Kind == "purge" {
		if err := state.verifyLibrary(); err != nil {
			return mutationReply{}, err
		}
		if err := state.rejectNestedMounts(); err != nil {
			return mutationReply{}, err
		}
		if request.Kind != "purge" && (request.Path == state.libraryRoot || state.libraryRoot != "" && !strings.HasPrefix(request.Path, state.libraryRoot+"/")) {
			return mutationReply{}, deletion.ErrInvalid
		}
	}
	switch request.Kind {
	case "probe":
		return mutationReply{}, nil
	case "inspect":
		identity, _, err := state.inspect(request.Path)
		return mutationReply{Identity: identity}, err
	case "inspect_trash":
		identity, _, err := state.inspect(mutationPayload(request.Instance, request.Entry))
		return mutationReply{Identity: identity}, err
	case "bind":
		state.closeRemoval()
		state.libraryRoot, state.libraryIdentity = request.Path, request.Expected
		return mutationReply{}, state.verifyLibrary()
	case "prepare":
		return mutationReply{}, state.prepareTrash(request.Instance)
	case "trash", "restore":
		return mutationReply{}, state.move(request)
	case "remove", "purge":
		relative := request.Path
		if request.Kind == "purge" {
			relative = mutationPayload(request.Instance, request.Entry)
		}
		state.dirty = make(map[string]struct{})
		reply, err := state.removeBatch(ctx, relative, request.Expected)
		if syncErr := state.syncDirty(); err == nil {
			err = syncErr
		}
		if err != nil {
			reply.Done = false
		}
		return reply, err
	default:
		return mutationReply{}, deletion.ErrInvalid
	}
}

func (state *mutationState) verifyLibrary() error {
	if state.libraryIdentity.Inode == 0 {
		return deletion.ErrInvalid
	}
	identity, _, err := state.inspect(state.libraryRoot)
	if err != nil || !state.libraryIdentity.Matches(identity) {
		return deletion.ErrSourceChanged
	}
	return nil
}

func mutationPayload(instance, entry string) string {
	return path.Join(deletion.TrashDirectory, instance, entry, "payload")
}

func (state *mutationState) inspect(relative string) (deletion.SourceIdentity, fs.FileMode, error) {
	file, err := state.root.openFile(relative, unix.O_PATH|unix.O_NOFOLLOW)
	if err != nil {
		return deletion.SourceIdentity{}, 0, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return deletion.SourceIdentity{}, 0, err
	}
	device, inode, ok := platformIdentity(info)
	if !ok {
		return deletion.SourceIdentity{}, 0, deletion.ErrWriteUnavailable
	}
	return deletion.SourceIdentity{Device: device, Inode: inode, Size: info.Size(), ModifiedNS: info.ModTime().UnixNano(), Directory: info.IsDir()}, info.Mode(), nil
}

func (state *mutationState) prepareTrash(instance string) error {
	base := deletion.TrashDirectory
	if err := state.mkdir(base); err != nil {
		if !errors.Is(err, fs.ErrExist) {
			return err
		}
		_, mode, err := state.inspect(path.Join(base, instance))
		if err != nil || !mode.IsDir() {
			return deletion.ErrConflict
		}
		return nil
	}
	return state.mkdir(path.Join(base, instance))
}

func (state *mutationState) mkdir(relative string) error {
	parentPath, name := splitParent(relative)
	parent, err := state.root.openFile(parentPath, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW)
	if err != nil {
		return err
	}
	defer parent.Close()
	if err := unix.Mkdirat(int(parent.Fd()), name, 0o700); err != nil {
		return err
	}
	return unix.Fsync(int(parent.Fd()))
}

func (state *mutationState) move(request mutationRequest) error {
	source, destination := request.Path, mutationPayload(request.Instance, request.Entry)
	if request.Kind == "restore" {
		source, destination = destination, source
	}
	identity, mode, err := state.inspect(source)
	if err != nil {
		return err
	}
	if !mode.IsRegular() && !mode.IsDir() || !request.Expected.Matches(identity) {
		return deletion.ErrSourceChanged
	}
	if request.Kind == "trash" {
		if err := state.mkdir(path.Dir(destination)); err != nil && !errors.Is(err, fs.ErrExist) {
			return err
		}
	}
	oldParentPath, oldName := splitParent(source)
	newParentPath, newName := splitParent(destination)
	oldParent, err := state.root.openFile(oldParentPath, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW)
	if err != nil {
		return err
	}
	defer oldParent.Close()
	newParent, err := state.root.openFile(newParentPath, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW)
	if err != nil {
		return err
	}
	defer newParent.Close()
	if err := unix.Renameat2(int(oldParent.Fd()), oldName, int(newParent.Fd()), newName, unix.RENAME_NOREPLACE); err != nil {
		return err
	}
	current, _, err := state.inspect(destination)
	if err != nil {
		return err
	}
	if !request.Expected.Matches(current) {
		return deletion.ErrSourceChanged
	}
	if err := unix.Fsync(int(oldParent.Fd())); err != nil {
		return err
	}
	return unix.Fsync(int(newParent.Fd()))
}

func (state *mutationState) rejectNestedMounts() error {
	rootPath, err := os.Readlink("/proc/self/fd/" + strconv.Itoa(int(state.root.boundary.Fd())))
	if err != nil {
		return deletion.ErrWriteUnavailable
	}
	file, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return deletion.ErrWriteUnavailable
	}
	defer file.Close()
	reader := &io.LimitedReader{R: file, N: (4 << 20) + 1}
	scanner := bufio.NewScanner(reader)
	escapes := strings.NewReplacer(`\040`, " ", `\011`, "\t", `\012`, "\n", `\134`, `\`)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 5 {
			return deletion.ErrWriteUnavailable
		}
		if strings.HasPrefix(escapes.Replace(fields[4]), strings.TrimSuffix(rootPath, "/")+"/") {
			return deletion.ErrWriteUnavailable
		}
	}
	if scanner.Err() != nil || reader.N == 0 {
		return deletion.ErrWriteUnavailable
	}
	return nil
}

type mutationFrame struct {
	path     string
	file     *os.File
	identity deletion.SourceIdentity
	pending  []fs.DirEntry
	index    int
}

type mutationRemoval struct {
	path     string
	expected deletion.SourceIdentity
	stack    []*mutationFrame
}

func (state *mutationState) closeRemoval() {
	if state.removal == nil {
		return
	}
	for _, frame := range state.removal.stack {
		_ = frame.file.Close()
	}
	state.removal = nil
}

func (state *mutationState) removeBatch(ctx context.Context, relative string, expected deletion.SourceIdentity) (mutationReply, error) {
	current, mode, err := state.inspect(relative)
	if err != nil {
		return mutationReply{}, err
	}
	if !expected.Matches(current) || !mode.IsDir() && !mode.IsRegular() {
		return mutationReply{}, deletion.ErrSourceChanged
	}
	if !mode.IsDir() {
		err := state.unlink(relative, 0)
		if err != nil {
			return mutationReply{}, err
		}
		return mutationReply{Removed: 1, Done: true}, nil
	}
	if state.removal == nil || state.removal.path != relative {
		state.closeRemoval()
		frame, err := state.openRemovalFrame(relative, current)
		if err != nil {
			return mutationReply{}, err
		}
		state.removal = &mutationRemoval{path: relative, expected: expected, stack: []*mutationFrame{frame}}
	}
	result := mutationReply{}
	for result.Removed < walkBatchSize && len(state.removal.stack) > 0 {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		stack := state.removal.stack
		frame := stack[len(stack)-1]
		identity, _, err := state.inspect(frame.path)
		if err != nil {
			return result, err
		}
		if !frame.identity.Matches(identity) {
			return result, deletion.ErrSourceChanged
		}
		if frame.index >= len(frame.pending) {
			frame.pending, err = frame.file.ReadDir(walkBatchSize)
			frame.index = 0
			if err != nil && !errors.Is(err, io.EOF) {
				return result, err
			}
			if len(frame.pending) == 0 {
				if err := state.unlink(frame.path, unix.AT_REMOVEDIR); err != nil {
					return result, err
				}
				_ = frame.file.Close()
				state.removal.stack = stack[:len(stack)-1]
				result.Removed++
				continue
			}
		}
		entry := frame.pending[frame.index]
		frame.index++
		childPath := path.Join(frame.path, entry.Name())
		identity, mode, err := state.inspect(childPath)
		if err != nil {
			return result, err
		}
		if mode.IsDir() {
			if len(stack) >= 128 {
				return result, deletion.ErrSourceIO
			}
			child, err := state.openRemovalFrame(childPath, identity)
			if err != nil {
				return result, err
			}
			state.removal.stack = append(stack, child)
		} else {
			if err := state.unlink(childPath, 0); err != nil {
				return result, err
			}
			result.Removed++
		}
	}
	result.Done = len(state.removal.stack) == 0
	if result.Done {
		state.closeRemoval()
	}
	return result, nil
}

func (state *mutationState) openRemovalFrame(relative string, identity deletion.SourceIdentity) (*mutationFrame, error) {
	file, err := state.root.openFile(relative, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	device, inode, _ := platformIdentity(info)
	if device != identity.Device || inode != identity.Inode {
		_ = file.Close()
		return nil, deletion.ErrSourceChanged
	}
	return &mutationFrame{path: relative, file: file, identity: identity}, nil
}

func (state *mutationState) unlink(relative string, flags int) error {
	parentPath, name := splitParent(relative)
	parent, err := state.root.openFile(parentPath, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW)
	if err != nil {
		return err
	}
	defer parent.Close()
	if err := unix.Unlinkat(int(parent.Fd()), name, flags); err != nil {
		return err
	}
	state.dirty[parentPath] = struct{}{}
	return nil
}

func (state *mutationState) syncDirty() error {
	for relative := range state.dirty {
		parent, err := state.root.openFile(relative, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		err = unix.Fsync(int(parent.Fd()))
		_ = parent.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func mutationOperationError(err error) error {
	switch {
	case errors.Is(err, deletion.ErrInvalid):
		return deletion.ErrInvalid
	case errors.Is(err, deletion.ErrSourceChanged), errors.Is(err, ErrChanged), errors.Is(err, ErrSymlink):
		return deletion.ErrSourceChanged
	case errors.Is(err, fs.ErrNotExist):
		return deletion.ErrSourceMissing
	case errors.Is(err, fs.ErrExist), errors.Is(err, deletion.ErrConflict):
		return deletion.ErrConflict
	case errors.Is(err, fs.ErrPermission), errors.Is(err, unix.EROFS):
		return deletion.ErrPermission
	case errors.Is(err, deletion.ErrWriteUnavailable), errors.Is(err, ErrKernelBoundaryUnavailable), errors.Is(err, ErrCrossDevice):
		return deletion.ErrWriteUnavailable
	default:
		return deletion.ErrSourceIO
	}
}
