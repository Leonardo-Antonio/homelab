package settings

import (
	"context"
	"path/filepath"
	"testing"

	"homelab/backend/internal/database"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewService(NewRepository(db))
}

func TestGetReturnsDefaultsWhenEmpty(t *testing.T) {
	got, err := newTestService(t).Get(context.Background())
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	want := Default()
	if got.Theme != want.Theme || got.Language != want.Language || got.Font != want.Font {
		t.Fatalf("got %+v, want defaults %+v", got, want)
	}
	for _, id := range KnownModules {
		if !got.Modules[id] {
			t.Fatalf("module %q should default to enabled", id)
		}
	}
}

func TestUpdateRoundTrips(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	in := Settings{
		Theme:    "dark",
		Language: "en",
		Font:     "mono",
		Modules:  map[string]bool{"camera": false, "terminal": false},
	}
	saved, err := svc.Update(ctx, in)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if saved.Theme != "dark" || saved.Language != "en" || saved.Font != "mono" {
		t.Fatalf("saved mismatch: %+v", saved)
	}
	// Explicitly disabled modules persist; unmentioned ones default to enabled.
	if saved.Modules["camera"] || saved.Modules["terminal"] {
		t.Fatalf("disabled modules should stay false: %+v", saved.Modules)
	}
	if !saved.Modules["storage"] || !saved.Modules["notes"] {
		t.Fatalf("unmentioned modules should default to enabled: %+v", saved.Modules)
	}

	// A fresh read returns the persisted document.
	reloaded, err := svc.Get(ctx)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Theme != "dark" || reloaded.Modules["camera"] {
		t.Fatalf("not persisted: %+v", reloaded)
	}
}

func TestModuleOrderNormalization(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	// Client sends a partial, out-of-order list with an unknown id and a dup.
	saved, err := svc.Update(ctx, Settings{
		Theme: "light", Language: "es", Font: "sans",
		ModuleOrder: []string{"storage", "notes", "storage", "hacking"},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	// Requested ids keep their order; the unknown id and dup are dropped; the
	// remaining known modules are appended in default order.
	want := []string{"storage", "notes", "clipboard", "photos", "camera", "terminal"}
	if len(saved.ModuleOrder) != len(want) {
		t.Fatalf("order length = %d, want %d: %v", len(saved.ModuleOrder), len(want), saved.ModuleOrder)
	}
	for i := range want {
		if saved.ModuleOrder[i] != want[i] {
			t.Fatalf("order = %v, want %v", saved.ModuleOrder, want)
		}
	}

	// And it is a complete permutation: every known module appears once.
	seen := map[string]int{}
	for _, id := range saved.ModuleOrder {
		seen[id]++
	}
	for _, id := range KnownModules {
		if seen[id] != 1 {
			t.Fatalf("module %q appears %d times", id, seen[id])
		}
	}
}

func TestDefaultsIncludeModuleOrder(t *testing.T) {
	got, err := newTestService(t).Get(context.Background())
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if len(got.ModuleOrder) != len(KnownModules) {
		t.Fatalf("default order = %v, want all known modules", got.ModuleOrder)
	}
	for i, id := range KnownModules {
		if got.ModuleOrder[i] != id {
			t.Fatalf("default order[%d] = %q, want %q", i, got.ModuleOrder[i], id)
		}
	}
}

func TestUpdateRejectsInvalidValues(t *testing.T) {
	svc := newTestService(t)
	cases := []Settings{
		{Theme: "neon", Language: "es", Font: "sans"},
		{Theme: "dark", Language: "fr", Font: "sans"},
		{Theme: "dark", Language: "es", Font: "comic"},
	}
	for _, c := range cases {
		if _, err := svc.Update(context.Background(), c); err != ErrInvalidSettings {
			t.Fatalf("%+v: err = %v, want ErrInvalidSettings", c, err)
		}
	}
}

func TestUpdateDropsUnknownModules(t *testing.T) {
	svc := newTestService(t)
	saved, err := svc.Update(context.Background(), Settings{
		Theme: "light", Language: "es", Font: "sans",
		Modules: map[string]bool{"hacking": true, "photos": false},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if _, ok := saved.Modules["hacking"]; ok {
		t.Fatalf("unknown module should be dropped: %+v", saved.Modules)
	}
	if saved.Modules["photos"] {
		t.Fatalf("photos should be disabled: %+v", saved.Modules)
	}
}
