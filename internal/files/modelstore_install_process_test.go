package files

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/aimodel"
	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
)

const (
	managedInstallKillHelperEnv = "FOLIOPATH_MANAGED_INSTALL_KILL_HELPER"
	managedInstallKillDataEnv   = "FOLIOPATH_MANAGED_INSTALL_KILL_DATA"
	managedInstallKillRootEnv   = "FOLIOPATH_MANAGED_INSTALL_KILL_ROOT"
)

func TestManagedInstallRecoversPublishedOrphanAfterProcessKill(t *testing.T) {
	root := t.TempDir()
	dataPath := filepath.Join(root, "foliopath.db")
	modelRoot := filepath.Join(root, "models")
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestManagedInstallKillHelper$")
	command.Env = append(os.Environ(),
		managedInstallKillHelperEnv+"=1",
		managedInstallKillDataEnv+"="+dataPath,
		managedInstallKillRootEnv+"="+modelRoot,
	)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || strings.TrimSpace(line) != "PUBLISHED_BEFORE_REGISTRATION" {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf("install helper output=%q err=%v stderr=%q", line, err, stderr.String())
	}
	if err := command.Process.Kill(); err != nil {
		_ = command.Wait()
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("managed install helper unexpectedly exited successfully")
	}

	store, err := sqlitestore.Open(context.Background(), dataPath, sqlitestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	operations, err := aimodel.NewOperationService(store, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if recovered, err := operations.RecoverInterrupted(context.Background()); err != nil || recovered != 1 {
		t.Fatalf("operation recovery=%d err=%v", recovered, err)
	}
	work, found, err := store.FindAIModelInstall(context.Background(), "install-process-kill")
	if err != nil || !found || work.Operation.State != aimodel.OperationFailed || work.Operation.ErrorCode != "operation_interrupted" {
		t.Fatalf("recovered install=%#v found=%v err=%v", work.Operation, found, err)
	}

	managed, err := NewManagedModelStore(modelRoot, 1<<30)
	if err != nil {
		t.Fatal(err)
	}
	report, err := managed.Reconcile(context.Background())
	if err != nil || report.KnownFinals != 1 || report.RemovedStaging != 0 || report.Truncated {
		t.Fatalf("managed reconcile=%#v err=%v", report, err)
	}
	manifest, verified, _ := managedInstallKillFixture()
	catalog, err := aimodel.NewCatalog([]aimodel.CatalogEntry{{
		Manifest: manifest, ContentHash: verified.ContentHash, RuntimeArchitectures: []string{runtime.GOARCH},
	}})
	if err != nil {
		t.Fatal(err)
	}
	models, err := aimodel.NewService(store, nil, func() (string, error) { return "aim_recovered_install", nil })
	if err != nil {
		t.Fatal(err)
	}
	orphans, err := aimodel.NewManagedOrphanService(models, catalog, managed, runtime.GOARCH)
	if err != nil {
		t.Fatal(err)
	}
	summary, err := orphans.Reconcile(context.Background(), report.FinalContentHashes, true)
	if err != nil || summary.Registered != 1 {
		t.Fatalf("orphan recovery=%#v err=%v", summary, err)
	}
	snapshot, err := models.List(context.Background())
	if err != nil || len(snapshot.Items) != 1 || snapshot.Items[0].State != aimodel.StateAvailable ||
		snapshot.Items[0].Active || snapshot.ActiveModelID != "" {
		t.Fatalf("recovered models=%#v err=%v", snapshot, err)
	}
}

func TestManagedInstallKillHelper(t *testing.T) {
	if os.Getenv(managedInstallKillHelperEnv) != "1" {
		return
	}
	ctx := context.Background()
	store, err := sqlitestore.Open(ctx, os.Getenv(managedInstallKillDataEnv), sqlitestore.Options{})
	if err != nil {
		fmt.Fprintf(os.Stdout, "ERROR:%v\n", err)
		os.Exit(2)
	}
	managed, err := NewManagedModelStore(os.Getenv(managedInstallKillRootEnv), 1<<30)
	if err != nil {
		fmt.Fprintf(os.Stdout, "ERROR:%v\n", err)
		os.Exit(2)
	}
	allowManagedPublishForTest(managed)
	manifest, verified, contents := managedInstallKillFixture()
	models, _ := aimodel.NewService(store, nil, func() (string, error) { return "aim_must_not_commit", nil })
	publisher := managedInstallBlockingPublisher{delegate: managed}
	installer, _ := aimodel.NewInstaller(models, managedInstallSource{contents: contents}, publisher)
	operations, _ := aimodel.NewOperationService(store, nil, nil)
	now := time.Now().UTC()
	candidate := aimodel.Candidate{ID: "aic_process_kill", Compatibility: "compatible", SourceIdentity: "source:process-kill",
		Manifest: manifest, Package: verified}
	work := aimodel.InstallWork{IdempotencyKey: "install-process-kill", CandidateID: candidate.ID,
		RequestHash: aimodel.InstallRequestHash(candidate.ID, aimodel.StorageManaged), Candidate: candidate,
		StorageMode: aimodel.StorageManaged, Operation: aimodel.Operation{ID: "aio_process_kill", Kind: aimodel.OperationModelInstall,
			State: aimodel.OperationQueued, Phase: aimodel.PhaseQueued, Revision: 1, CreatedAt: now, UpdatedAt: now}}
	if _, _, err := store.CreateAIModelInstall(ctx, work); err != nil {
		fmt.Fprintf(os.Stdout, "ERROR:%v\n", err)
		os.Exit(2)
	}
	worker, _ := aimodel.NewInstallWorker(store, installer, operations, make(chan struct{}), 10*time.Millisecond)
	if err := worker.Run(ctx); err != nil {
		fmt.Fprintf(os.Stdout, "ERROR:%v\n", err)
		os.Exit(2)
	}
	os.Exit(3)
}

type managedInstallSource struct{ contents map[string][]byte }

func (source managedInstallSource) OpenModelPackageFile(_ context.Context, _ string, name string) (io.ReadCloser, int64, error) {
	content, ok := source.contents[name]
	if !ok {
		return nil, 0, os.ErrNotExist
	}
	return io.NopCloser(bytes.NewReader(content)), int64(len(content)), nil
}
func (managedInstallSource) ValidateDirectModelSource(context.Context, string) error { return nil }

type managedInstallBlockingPublisher struct{ delegate aimodel.ManagedPublisher }

func (publisher managedInstallBlockingPublisher) PublishModelPackage(ctx context.Context, model aimodel.VerifiedPackage, manifest aimodel.Manifest, opener aimodel.PackageOpener) (string, error) {
	identity, err := publisher.delegate.PublishModelPackage(ctx, model, manifest, opener)
	if err != nil {
		return "", err
	}
	if identity != "managed:"+model.ContentHash {
		return "", aimodel.ErrRepositoryState
	}
	fmt.Fprintln(os.Stdout, "PUBLISHED_BEFORE_REGISTRATION")
	select {}
}

func managedInstallKillFixture() (aimodel.Manifest, aimodel.VerifiedPackage, map[string][]byte) {
	manifest, verified, contents := managedPublishKillFixture()
	verified.PackageID = "semantic-install-kill-v1"
	verified.Architecture = runtime.GOARCH
	manifest.PackageID = verified.PackageID
	return manifest, verified, contents
}
