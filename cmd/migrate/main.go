package main

import (
	"github.com/ridwanmuh3/simba-server/internal/config"
	"github.com/ridwanmuh3/simba-server/internal/entity"
)

// fksNeedingCascade lists foreign keys that must delete their children along
// with the parent. See step 7 for why existing databases need the repair.
var fksNeedingCascade = []struct {
	table     string
	column    string
	refTable  string
	refColumn string
}{
	{table: "invoice_items", column: "invoice_id", refTable: "invoices", refColumn: "id"},
	{table: "stock_tracks", column: "item_id", refTable: "items", refColumn: "id"},
}

// tablesNeedingDapur lists tables that require a dapur_id backfill when the
// column is added to a database that already has rows.
var tablesNeedingDapur = []string{
	"items",
	"stock_tracks",
	"finances",
	"activity_logs",
	"app_settings",
	"invoices",
}

func main() {
	viperConfig := config.NewViper()
	log := config.NewLogger(viperConfig)
	db := config.NewDB(viperConfig, log)

	log.Info("running schema migration...")

	// 1. Create dapurs table first so we can seed a default dapur.
	if err := db.AutoMigrate(&entity.Dapur{}); err != nil {
		log.Fatalf("migration failed (dapur): %v", err)
	}

	// 2. Ensure at least one dapur exists; use it as the backfill value.
	var defaultDapur entity.Dapur
	if err := db.First(&defaultDapur).Error; err != nil {
		defaultDapur = entity.Dapur{Name: "Dapur Utama", Description: "Dapur default hasil migrasi", IsActive: true}
		if err := db.Create(&defaultDapur).Error; err != nil {
			log.Fatalf("failed to seed default dapur: %v", err)
		}
		log.Infof("created default dapur with id=%d", defaultDapur.ID)
	}

	// 3. For each table that needs dapur_id, add it as nullable then backfill
	//    before enforcing NOT NULL. This avoids "column contains null values"
	//    errors on tables that already have rows.
	for _, table := range tablesNeedingDapur {
		// On an empty database the tables do not exist yet — step 5 creates them
		// with dapur_id already in place, so there is nothing to backfill.
		if !db.Migrator().HasTable(table) {
			log.Infof("skipping dapur_id backfill on %s: table does not exist yet", table)
			continue
		}

		if !db.Migrator().HasColumn(table, "dapur_id") {
			log.Infof("adding nullable dapur_id to %s", table)
			if err := db.Exec("ALTER TABLE " + table + " ADD COLUMN dapur_id bigint").Error; err != nil {
				log.Fatalf("failed to add dapur_id to %s: %v", table, err)
			}
			log.Infof("backfilling dapur_id=%d on %s", defaultDapur.ID, table)
			if err := db.Exec("UPDATE "+table+" SET dapur_id = ? WHERE dapur_id IS NULL", defaultDapur.ID).Error; err != nil {
				log.Fatalf("failed to backfill dapur_id on %s: %v", table, err)
			}
			log.Infof("setting NOT NULL on %s.dapur_id", table)
			if err := db.Exec("ALTER TABLE " + table + " ALTER COLUMN dapur_id SET NOT NULL").Error; err != nil {
				log.Fatalf("failed to set NOT NULL on %s.dapur_id: %v", table, err)
			}
		}
	}

	// 4. Drop initial_unit_price column from items if it exists — field removed.
	if db.Migrator().HasColumn("items", "initial_unit_price") {
		log.Info("dropping items.initial_unit_price")
		if err := db.Exec("ALTER TABLE items DROP COLUMN initial_unit_price").Error; err != nil {
			log.Fatalf("failed to drop initial_unit_price: %v", err)
		}
	}

	// 5. AutoMigrate all entities — adds missing columns, indexes, constraints.
	//    Runs after backfills so all pre-existing rows already have valid data.
	if err := db.AutoMigrate(
		&entity.User{},
		&entity.Item{},
		&entity.StockTracking{},
		&entity.Finance{},
		&entity.ActivityLog{},
		&entity.AppSetting{},
		&entity.Invoice{},
		&entity.InvoiceItem{},
		&entity.Notification{},
	); err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	// 6. Drop old single-column unique indexes on invoices — replaced by
	//    composite (stock_type, *) indexes to allow same number across stock types.
	oldIndexes := []string{
		"idx_invoices_invoice_number",
		"idx_invoice_po_number",
	}
	for _, idx := range oldIndexes {
		if db.Migrator().HasIndex(&entity.Invoice{}, idx) {
			if err := db.Migrator().DropIndex(&entity.Invoice{}, idx); err != nil {
				log.Warnf("could not drop index %s: %v", idx, err)
			}
		}
	}

	// 7. Recreate any foreign key that is missing ON DELETE CASCADE. The entity
	//    tags used to read `constraint:OnUpdate:CASCADE;OnDelete:CASCADE`, but
	//    GORM splits that tag on `;` and only honours the first option, so the
	//    schema ended up with `ON UPDATE CASCADE` alone. AutoMigrate does not
	//    alter an existing constraint, so affected databases keep the broken
	//    definition — and deleting a parent row fails while children reference
	//    it — until the constraint is dropped and recreated here.
	for _, fk := range fksNeedingCascade {
		var rows []struct{ Conname string }
		if err := db.Raw(`
			SELECT conname FROM pg_constraint
			WHERE contype = 'f'
			  AND conrelid = ?::regclass
			  AND confdeltype <> 'c'
			  AND pg_get_constraintdef(oid) LIKE ?
		`, fk.table, "%FOREIGN KEY ("+fk.column+")%").Scan(&rows).Error; err != nil {
			log.Fatalf("failed to inspect foreign keys on %s: %v", fk.table, err)
		}

		for _, row := range rows {
			log.Infof("recreating %s on %s with ON DELETE CASCADE", row.Conname, fk.table)
			if err := db.Exec("ALTER TABLE " + fk.table + " DROP CONSTRAINT " + row.Conname).Error; err != nil {
				log.Fatalf("failed to drop constraint %s: %v", row.Conname, err)
			}
			stmt := "ALTER TABLE " + fk.table + " ADD CONSTRAINT " + row.Conname +
				" FOREIGN KEY (" + fk.column + ") REFERENCES " + fk.refTable + "(" + fk.refColumn + ")" +
				" ON UPDATE CASCADE ON DELETE CASCADE"
			if err := db.Exec(stmt).Error; err != nil {
				log.Fatalf("failed to recreate constraint %s: %v", row.Conname, err)
			}
		}
	}

	// 8. Money/quantity columns: float8 -> NUMERIC. GORM entity tags now
	//    declare numeric(18,4)/numeric(18,2); AutoMigrate does not alter an
	//    existing column type, so convert explicitly (idempotent: only
	//    double-precision columns are touched). Values are exact-decimal
	//    in the DB from here on; Go still marshals float64 at the API edge
	//    while all chain math is integer-scaled (util.CalculateMAC).
	numericCols := []struct{ table, column string }{
		{"items", "initial_stock"}, {"items", "stock"}, {"items", "unit_price"},
		{"items", "total_price"}, {"items", "low_stock_threshold"},
		{"stock_tracks", "amount"}, {"stock_tracks", "previous_stock"},
		{"stock_tracks", "new_stock"}, {"stock_tracks", "unit_price"},
		{"stock_tracks", "total_price"},
		{"invoices", "grand_total"},
		{"invoice_items", "amount"}, {"invoice_items", "unit_price"},
		{"invoice_items", "total_price"}, {"invoice_items", "previous_stock"},
		{"invoice_items", "new_stock"},
	}
	for _, nc := range numericCols {
		var dataType string
		err := db.Raw(`SELECT data_type FROM information_schema.columns
			WHERE table_name = ? AND column_name = ?`, nc.table, nc.column).Scan(&dataType).Error
		if err != nil || dataType != "double precision" {
			continue
		}
		if err := db.Exec("ALTER TABLE " + nc.table + " ALTER COLUMN " + nc.column + " TYPE numeric(18,4) USING " + nc.column + "::numeric(18,4)").Error; err != nil {
			log.Fatalf("failed to convert %s.%s to numeric: %v", nc.table, nc.column, err)
		}
		log.Infof("converted %s.%s to numeric(18,4)", nc.table, nc.column)
	}

	// 9. Composite FK so a stock_track can never reference an item from a
	//    different dapur: (item_id, dapur_id) -> items(id, dapur_id).
	//    Guarded: skipped (with a warning) if orphan rows exist, so a
	//    dirty database fails soft instead of aborting the migration.
	var orphans int64
	if err := db.Raw(`SELECT COUNT(*) FROM stock_tracks st
		LEFT JOIN items i ON i.id = st.item_id AND i.dapur_id = st.dapur_id
		WHERE i.id IS NULL`).Scan(&orphans).Error; err != nil {
		log.Fatalf("failed to check stock_tracks orphans: %v", err)
	}
	if orphans > 0 {
		log.Warnf("skipping composite FK on stock_tracks: %d orphan row(s) with mismatched dapur_id", orphans)
	} else {
		var exists int64
		db.Raw(`SELECT COUNT(*) FROM pg_constraint WHERE conname = 'fk_stock_tracks_item_dapur'`).Scan(&exists)
		if exists == 0 {
			if err := db.Exec(`ALTER TABLE stock_tracks
				ADD CONSTRAINT fk_stock_tracks_item_dapur
				FOREIGN KEY (item_id, dapur_id) REFERENCES items(id, dapur_id)
				ON UPDATE CASCADE ON DELETE CASCADE`).Error; err != nil {
				log.Fatalf("failed to add composite FK on stock_tracks: %v", err)
			}
			log.Info("added composite FK stock_tracks(item_id, dapur_id) -> items(id, dapur_id)")
		}
	}

	// NOTE (deliberately not automated): Postgres Row-Level Security would
	// be the defence-in-depth layer on top of the Tenant() scope, but
	// enabling it requires every transaction to SET LOCAL app.dapur_id
	// first — rolling that out unguarded would lock the app out of its own
	// tables. Enable per-table once TenantTx sets the GUC (follow-up).

	log.Info("migration complete")
}
