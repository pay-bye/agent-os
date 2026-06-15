package migrations

import "testing"

func TestMigrationPlanAppliesSkipsAndRejectsMismatch(t *testing.T) {
	migrations := []Migration{
		{Name: "001.sql", Body: []byte("select 1;")},
		{Name: "002.sql", Body: []byte("select 2;")},
	}
	current := map[string]string{
		"001.sql": checksum(migrations[0].Body),
		"002.sql": "different",
	}

	plan, err := planMigrations(migrations, current)

	if err == nil {
		t.Fatal("expected checksum mismatch")
	}
	if len(plan.Apply) != 0 {
		t.Fatalf("plan = %+v", plan)
	}
}

func TestMigrationPlanSortsLexically(t *testing.T) {
	migrations := []Migration{
		{Name: "002.sql", Body: []byte("select 2;")},
		{Name: "001.sql", Body: []byte("select 1;")},
	}

	plan, err := planMigrations(migrations, map[string]string{})
	if err != nil {
		t.Fatal(err)
	}

	if plan.Apply[0].Name != "001.sql" || plan.Apply[1].Name != "002.sql" {
		t.Fatalf("apply order = %+v", plan.Apply)
	}
}

func TestNewMigratorLoadsEmbeddedMigrationsAndCopiesCustomSource(t *testing.T) {
	custom := []Migration{{Name: "999.sql", Body: []byte("select 9;")}}

	embedded := New(nil)
	migrator := New(nil, WithSearchPath("x01"), WithSource(custom))
	custom[0].Name = "changed.sql"

	if len(embedded.migrations) == 0 {
		t.Fatal("expected embedded migrations")
	}
	if migrator.searchPath != "x01" {
		t.Fatalf("search path = %q, want x01", migrator.searchPath)
	}
	if migrator.migrations[0].Name != "999.sql" {
		t.Fatalf("migration source was not copied: %+v", migrator.migrations)
	}
}
