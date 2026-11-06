package repository

import "gorm.io/gorm"

// Tenant scopes every query to one dapur (tenant). Every multi-tenant
// query must go through this scope (or TenantColumn for joined queries)
// so a forgotten dapur_id filter can no longer leak rows across tenants.
//
// Shared-database / shared-schema model: dapur_id is taken from the
// authenticated session server-side, never from an untrusted request
// body alone, and enforced here at the single place queries are built.
//
// dapurID == 0 is the super-admin cross-dapur view and intentionally
// unscoped; every handler/service path that calls Tenant with a tenant
// id validates dapur_id >= 1 (session-derived or `validate:"min=1"`).
func Tenant(dapurID uint) func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		if dapurID == 0 {
			return tx
		}
		return tx.Where("dapur_id = ?", dapurID)
	}
}

// TenantColumn is Tenant for joined queries where dapur_id is ambiguous:
// TenantColumn("items", id) adds `items.dapur_id = ?`.
func TenantColumn(table string, dapurID uint) func(*gorm.DB) *gorm.DB {
	return func(tx *gorm.DB) *gorm.DB {
		if dapurID == 0 {
			return tx
		}
		return tx.Where(table+".dapur_id = ?", dapurID)
	}
}
