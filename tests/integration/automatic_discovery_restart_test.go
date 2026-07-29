package integration

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/library"
	"github.com/HappyQuQu/foliopath/internal/scanner"
	sqlitestore "github.com/HappyQuQu/foliopath/internal/store/sqlite"
)

const (
	reconcileKillHelperEnv   = "FOLIOPATH_RECONCILE_KILL_HELPER"
	reconcileKillDatabaseEnv = "FOLIOPATH_RECONCILE_KILL_DATABASE"
)

func TestAutomaticDiscoveryRecoversClaimAcrossDatabaseRestart(t *testing.T) {
	ctx := context.Background()
	now := time.UnixMilli(10_000)
	filename := filepath.Join(t.TempDir(), "automatic-discovery-restart.db")
	options := sqlitestore.Options{Now: func() time.Time { return now }}

	firstStore, err := sqlitestore.Open(ctx, filename, options)
	if err != nil {
		t.Fatal(err)
	}
	libraries, err := library.NewService(firstStore)
	if err != nil {
		_ = firstStore.Close()
		t.Fatal(err)
	}
	item, err := libraries.Create(ctx, "Archive", "archive")
	if err != nil {
		_ = firstStore.Close()
		t.Fatal(err)
	}
	enqueued, err := firstStore.EnqueueReconcile(
		ctx,
		item.ID,
		"album",
		scanner.ReconcileDebounce,
		scanner.ReconcileMaximumDebounce,
	)
	if err != nil {
		_ = firstStore.Close()
		t.Fatal(err)
	}
	now = now.Add(time.Second)
	claimed, found, err := firstStore.ClaimNextReconcile(ctx, 100*time.Millisecond)
	if err != nil || !found || claimed.ID != enqueued.ID ||
		claimed.AttemptCount != 1 {
		_ = firstStore.Close()
		t.Fatalf("first claim = %#v found=%t err=%v", claimed, found, err)
	}

	// Closing without completing the running claim models the durable state
	// left behind when the process disappears after claim and before commit.
	if err := firstStore.Close(); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Second)

	restarted, err := sqlitestore.Open(ctx, filename, options)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = restarted.Close() })
	summary, err := restarted.RecoverExpiredReconciles(ctx)
	if err != nil || summary.Requeued != 1 || summary.Interrupted != 0 {
		t.Fatalf("restart recovery = %#v err=%v", summary, err)
	}
	if _, found, err := restarted.ClaimNextReconcile(
		ctx,
		time.Minute,
	); err != nil || found {
		t.Fatalf("claim before restart backoff found=%t err=%v", found, err)
	}
	now = now.Add(2 * time.Second)
	reclaimed, found, err := restarted.ClaimNextReconcile(ctx, time.Minute)
	if err != nil || !found ||
		reclaimed.ID != enqueued.ID ||
		reclaimed.RequestedRevision != enqueued.RequestedRevision ||
		reclaimed.AttemptCount != 2 {
		t.Fatalf("reclaimed job = %#v found=%t err=%v", reclaimed, found, err)
	}
}

func TestAutomaticDiscoveryRecoversClaimAfterProcessKill(t *testing.T) {
	ctx := context.Background()
	filename := filepath.Join(t.TempDir(), "automatic-discovery-kill.db")
	store, err := sqlitestore.Open(ctx, filename, sqlitestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	libraries, err := library.NewService(store)
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	item, err := libraries.Create(ctx, "Archive", "archive")
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	enqueued, err := store.EnqueueReconcile(
		ctx,
		item.ID,
		"album",
		time.Millisecond,
		time.Millisecond,
	)
	if err != nil {
		_ = store.Close()
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(5 * time.Millisecond)

	helperContext, cancelHelper := context.WithTimeout(ctx, 5*time.Second)
	defer cancelHelper()
	command := exec.CommandContext(
		helperContext,
		os.Args[0],
		"-test.run=^TestAutomaticDiscoveryClaimKillHelper$",
	)
	command.Env = append(
		os.Environ(),
		reconcileKillHelperEnv+"=1",
		reconcileKillDatabaseEnv+"="+filename,
	)
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		_ = command.Wait()
		t.Fatalf("read claim helper: %v", err)
	}
	fields := strings.Split(strings.TrimSpace(line), ":")
	if len(fields) != 2 || fields[0] != "CLAIMED" {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf("unexpected claim helper output %q", line)
	}
	claimedID, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil || claimedID != enqueued.ID {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf("claim helper job = %q, want %d", fields[1], enqueued.ID)
	}
	if err := command.Process.Kill(); err != nil {
		_ = command.Wait()
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("claim helper unexpectedly exited successfully")
	}

	time.Sleep(150 * time.Millisecond)
	restarted, err := sqlitestore.Open(ctx, filename, sqlitestore.Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = restarted.Close() })
	summary, err := restarted.RecoverExpiredReconciles(ctx)
	if err != nil || summary.Requeued != 1 || summary.Interrupted != 0 {
		t.Fatalf("post-kill recovery = %#v err=%v", summary, err)
	}
	if _, found, err := restarted.ClaimNextReconcile(
		ctx,
		time.Minute,
	); err != nil || found {
		t.Fatalf("claim before post-kill backoff found=%t err=%v", found, err)
	}
	time.Sleep(1100 * time.Millisecond)
	reclaimed, found, err := restarted.ClaimNextReconcile(ctx, time.Minute)
	if err != nil || !found ||
		reclaimed.ID != enqueued.ID ||
		reclaimed.RequestedRevision != enqueued.RequestedRevision ||
		reclaimed.AttemptCount != 2 {
		t.Fatalf("post-kill reclaimed job = %#v found=%t err=%v", reclaimed, found, err)
	}
}

func TestAutomaticDiscoveryClaimKillHelper(t *testing.T) {
	if os.Getenv(reconcileKillHelperEnv) != "1" {
		return
	}
	store, err := sqlitestore.Open(
		context.Background(),
		os.Getenv(reconcileKillDatabaseEnv),
		sqlitestore.Options{},
	)
	if err != nil {
		fmt.Fprintf(os.Stdout, "ERROR:%v\n", err)
		os.Exit(2)
	}
	claimed, found, err := store.ClaimNextReconcile(
		context.Background(),
		100*time.Millisecond,
	)
	if err != nil || !found {
		fmt.Fprintf(os.Stdout, "ERROR:found=%t err=%v\n", found, err)
		os.Exit(2)
	}
	fmt.Fprintf(os.Stdout, "CLAIMED:%d\n", claimed.ID)
	for {
		time.Sleep(time.Hour)
	}
}
