package hexcore_test

import (
	"context"
	"os"
	"testing"
	"time"

	hexcore "github.com/2chevskii/go-hexcore"
)

// TestLiveVDSProvisioning exercises one standard VDS before the broader write
// suite. It makes one creation attempt, including when balance is insufficient.
func TestLiveVDSProvisioning(t *testing.T) {
	if os.Getenv("HEXCORE_LIVE_WRITES") != "1" {
		t.Skip("set HEXCORE_LIVE_WRITES=1 as well as HEXCORE_LIVE=1 to create one disposable VDS")
	}
	c := liveClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	liveVDSFixture(t, ctx, c)
}

// TestLiveVDSActions keeps lifecycle verification independent of additional
// disk provisioning. It creates and cleans up exactly one standard VDS.
func TestLiveVDSActions(t *testing.T) {
	if os.Getenv("HEXCORE_LIVE_WRITES") != "1" {
		t.Skip("set HEXCORE_LIVE_WRITES=1 as well as HEXCORE_LIVE=1 to check VDS lifecycle actions")
	}
	c := liveClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	vds := liveVDSFixture(t, ctx, c)
	liveVDSActions(t, ctx, c, vds.ID)
}

func liveVDSFixture(t *testing.T, ctx context.Context, c *hexcore.Client) *hexcore.VDS {
	t.Helper()
	req := hexcore.CreateVDSRequest{
		Name:     liveResourcePrefix(t) + "-probe",
		Size:     liveRequired(t, "HEXCORE_SIZE"),
		Image:    liveRequired(t, "HEXCORE_IMAGE"),
		Region:   liveRequired(t, "HEXCORE_REGION"),
		Password: "Aa1!" + randomSuffix(t),
	}
	var createdID string
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cleanupCancel()
		ids := make(map[string]bool)
		if createdID != "" {
			ids[createdID] = true
		}
		// Reconcile the unique name even if the POST response was lost or failed
		// decoding. Never select preexisting account resources for cleanup.
		items, err := c.VDS.List(cleanupCtx)
		if err != nil {
			t.Errorf("provisioning cleanup discovery failed; inspect name %s: %v", req.Name, err)
		}
		for _, item := range items {
			if item.Name == req.Name {
				ids[item.ID] = true
			}
		}
		if len(ids) == 0 && err == nil {
			t.Log("no VDS created by this provisioning check remains in the account")
		}
		for id := range ids {
			if err := liveDestroyVDS(cleanupCtx, c, id); err != nil {
				t.Errorf("test VDS %s cleanup returned an error; verify removal: %v", id, err)
				continue
			}
		}
	})

	created, err := c.VDS.Create(ctx, req)
	if err != nil {
		t.Fatalf("create standard VDS: %v", err)
	}
	createdID = created.ID
	t.Logf("created disposable VDS ID=%s name=%s", createdID, req.Name)
	if err := liveVDSState(ctx, c, createdID, "Running"); err != nil {
		t.Fatal(err)
	}
	current, err := c.VDS.Get(ctx, createdID)
	if err != nil {
		t.Fatal(err)
	}
	if current.ID != createdID || current.Name != req.Name {
		t.Fatal("retrieved VDS does not match the created test resource")
	}
	return current
}
