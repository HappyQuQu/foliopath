package library

import (
	"context"
	"errors"
	"testing"
)

type lifecycleRepositoryStub struct {
	findCreate func(context.Context, [32]byte, [32]byte) (CreateResult, bool, error)
	create     func(context.Context, CreateCommand) (CreateResult, error)
	list       func(context.Context, ListParams) ([]Details, error)
}

func (stub lifecycleRepositoryStub) FindCreateReplay(
	ctx context.Context,
	key, request [32]byte,
) (CreateResult, bool, error) {
	return stub.findCreate(ctx, key, request)
}

func (stub lifecycleRepositoryStub) CreateLibraryWithScan(
	ctx context.Context,
	command CreateCommand,
) (CreateResult, error) {
	return stub.create(ctx, command)
}

func (stub lifecycleRepositoryStub) ListLibraryPage(
	ctx context.Context,
	params ListParams,
) ([]Details, error) {
	return stub.list(ctx, params)
}

func (lifecycleRepositoryStub) GetLibraryDetails(context.Context, int64) (Details, error) {
	return Details{}, errors.New("unexpected GetLibraryDetails")
}

func (lifecycleRepositoryStub) RenameLibraryIfRevision(context.Context, RenameCommand) (Details, error) {
	return Details{}, errors.New("unexpected RenameLibraryIfRevision")
}

func (lifecycleRepositoryStub) RequestLibraryRemoval(context.Context, RemoveCommand) (RemoveResult, error) {
	return RemoveResult{}, errors.New("unexpected RequestLibraryRemoval")
}

func (lifecycleRepositoryStub) GetLibraryRemoval(context.Context, int64) (Removal, error) {
	return Removal{}, errors.New("unexpected GetLibraryRemoval")
}

type rootValidatorStub struct {
	calls int
	err   error
}

func (stub *rootValidatorStub) ValidateLibraryRoot(context.Context, string) error {
	stub.calls++
	return stub.err
}

type wakeStub struct{ calls int }

func (stub *wakeStub) Wake() { stub.calls++ }

func TestLifecycleCreateValidatesRootBeforeCommitAndWakesAfterCommit(t *testing.T) {
	roots := &rootValidatorStub{}
	waker := &wakeStub{}
	repository := lifecycleRepositoryStub{
		findCreate: func(context.Context, [32]byte, [32]byte) (CreateResult, bool, error) {
			return CreateResult{}, false, nil
		},
		create: func(_ context.Context, command CreateCommand) (CreateResult, error) {
			if roots.calls != 1 {
				t.Fatalf("create observed root validation calls = %d, want 1", roots.calls)
			}
			if command.Name != "Family" || command.NameKey != "family" ||
				command.RootRelativePath != "family" {
				t.Fatalf("normalized command = %#v", command)
			}
			return CreateResult{
				Library: Details{Library: Library{ID: 1, Revision: 1}},
				Scan:    Scan{ID: 2, LibraryID: 1},
			}, nil
		},
		list: func(context.Context, ListParams) ([]Details, error) { return nil, nil },
	}
	service, err := NewLifecycleService(repository, roots, waker, &wakeStub{}, LifecycleOptions{
		CursorKey: make([]byte, 32),
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(context.Background(), " Family ", "family", "create-key-1"); err != nil {
		t.Fatal(err)
	}
	if waker.calls != 1 {
		t.Fatalf("wake calls = %d, want 1", waker.calls)
	}
}

func TestLifecycleCreateReplaySkipsChangedFilesystemAndWorkerWake(t *testing.T) {
	roots := &rootValidatorStub{err: ErrRootUnavailable}
	waker := &wakeStub{}
	want := CreateResult{
		Library:  Details{Library: Library{ID: 1, Revision: 1}},
		Scan:     Scan{ID: 2, LibraryID: 1},
		Replayed: true,
	}
	repository := lifecycleRepositoryStub{
		findCreate: func(context.Context, [32]byte, [32]byte) (CreateResult, bool, error) {
			return want, true, nil
		},
		create: func(context.Context, CreateCommand) (CreateResult, error) {
			return CreateResult{}, errors.New("unexpected create")
		},
		list: func(context.Context, ListParams) ([]Details, error) { return nil, nil },
	}
	service, err := NewLifecycleService(repository, roots, waker, &wakeStub{}, LifecycleOptions{
		CursorKey: make([]byte, 32),
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := service.Create(context.Background(), "Family", "family", "create-key-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Library.ID != want.Library.ID || roots.calls != 0 || waker.calls != 0 {
		t.Fatalf("replay = %#v, root calls = %d, wake calls = %d", got, roots.calls, waker.calls)
	}
}

func TestLifecycleListCursorIsBoundedAndIntegrityProtected(t *testing.T) {
	call := 0
	repository := lifecycleRepositoryStub{
		findCreate: func(context.Context, [32]byte, [32]byte) (CreateResult, bool, error) {
			return CreateResult{}, false, nil
		},
		create: func(context.Context, CreateCommand) (CreateResult, error) {
			return CreateResult{}, nil
		},
		list: func(_ context.Context, params ListParams) ([]Details, error) {
			call++
			if call == 1 {
				if params.After != nil || params.Limit != 2 {
					t.Fatalf("first params = %#v", params)
				}
				return []Details{
					{Library: Library{ID: 1, Name: "Album 1"}},
					{Library: Library{ID: 2, Name: "Album 2"}},
				}, nil
			}
			if params.After == nil || params.After.ID != 1 {
				t.Fatalf("cursor params = %#v", params)
			}
			return nil, nil
		},
	}
	service, err := NewLifecycleService(
		repository,
		&rootValidatorStub{},
		&wakeStub{},
		&wakeStub{},
		LifecycleOptions{CursorKey: make([]byte, 32)},
	)
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.List(context.Background(), "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Items) != 1 || first.NextCursor == "" {
		t.Fatalf("first page = %#v", first)
	}
	if _, err := service.List(context.Background(), first.NextCursor+"x", 1); !errors.Is(
		err,
		ErrInvalidLibraryCursor,
	) {
		t.Fatalf("tampered cursor error = %v", err)
	}
	if _, err := service.List(context.Background(), first.NextCursor, 1); err != nil {
		t.Fatalf("valid cursor error = %v", err)
	}
}
