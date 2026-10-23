package sqlite

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/HappyQuQu/foliopath/internal/resourcecontrol"
	"github.com/HappyQuQu/foliopath/internal/scanner"
)

type unopenedScanWalker struct {
	*fakeWalker
	t *testing.T
}

func (w unopenedScanWalker) CaptureRoot(context.Context, string) (scanner.RootIdentity, error) {
	w.t.Error("cancelled scan must not open its root")
	return scanner.RootIdentity{}, errors.New("unexpected media access")
}

func TestScanCancellationBeforeAdmissionPreservesCatalog(t *testing.T) {
	for _, limited := range []bool{false, true} {
		for _, deadline := range []bool{false, true} {
			name := "entry"
			if limited {
				name = "resource-wait"
			}
			if deadline {
				name += "/deadline"
			} else {
				name += "/cancelled"
			}
			t.Run(name, func(t *testing.T) {
				store, _ := openTestStore(t)
				library := seedBrowseCatalog(t, store)
				var beforeAssets, beforeDirectories, beforeGeneration int64
				if err := store.db.QueryRow(`SELECT (SELECT COUNT(*) FROM assets WHERE library_id=?),(SELECT COUNT(*) FROM directories WHERE library_id=?),current_generation FROM libraries WHERE id=?`, library, library, library).Scan(&beforeAssets, &beforeDirectories, &beforeGeneration); err != nil {
					t.Fatal(err)
				}
				if _, err := store.AdmitFullScan(context.Background(), library, scanner.TriggerManual); err != nil {
					t.Fatal(err)
				}
				run, found, err := store.ClaimNextFullScan(context.Background(), time.Minute)
				if err != nil || !found {
					t.Fatal(err)
				}
				service, err := scanner.NewService(store, scanner.Config{})
				if err != nil {
					t.Fatal(err)
				}
				processor, err := scanner.NewClaimedProcessor(service, unopenedScanWalker{&fakeWalker{}, t})
				if err != nil {
					t.Fatal(err)
				}
				var process resourcecontrol.Processor[scanner.ScanRun] = processor
				if limited {
					controller, _ := resourcecontrol.NewController(resourcecontrol.Limits{Background: 1, Content: 1})
					release, err := controller.AcquireBackground(context.Background())
					if err != nil {
						t.Fatal(err)
					}
					defer release()
					process, err = resourcecontrol.LimitBackground(controller, processor)
					if err != nil {
						t.Fatal(err)
					}
				}
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				wantErr := context.Canceled
				if deadline {
					ctx, cancel = context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
					defer cancel()
					wantErr = context.DeadlineExceeded
				}
				if err := process.Process(ctx, run); !errors.Is(err, wantErr) {
					t.Fatal(err)
				}
				var state, phase, libraryState string
				var leases, assets, directories, generation int64
				if err := store.db.QueryRow(`SELECT status,phase,lease_expires_at_ms IS NOT NULL FROM scan_runs WHERE id=?`, run.ID).Scan(&state, &phase, &leases); err != nil {
					t.Fatal(err)
				}
				if state != "cancelled" || phase != "completed" || leases != 0 {
					t.Fatalf("scan=%s/%s lease=%d", state, phase, leases)
				}
				if err := store.db.QueryRow(`SELECT (SELECT COUNT(*) FROM assets WHERE library_id=?),(SELECT COUNT(*) FROM directories WHERE library_id=?),current_generation,status FROM libraries WHERE id=?`, library, library, library).Scan(&assets, &directories, &generation, &libraryState); err != nil {
					t.Fatal(err)
				}
				if assets != beforeAssets || directories != beforeDirectories || generation != beforeGeneration || libraryState != "ready" {
					t.Fatalf("catalog changed: %d/%d/%d/%s", assets, directories, generation, libraryState)
				}
				if _, err := store.AdmitFullScan(context.Background(), library, scanner.TriggerManual); err != nil {
					t.Fatalf("retry admission: %v", err)
				}
			})
		}
	}
}
