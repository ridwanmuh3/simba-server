package util

import (
	"testing"

	"github.com/go-playground/validator/v10"
)

type testStruct struct {
	Name  string `validate:"required,min=2"`
	Email string `validate:"required,email"`
	Age   int    `validate:"required,min=1,max=150"`
}

func newTestValidator() *validator.Validate {
	v := validator.New()
	_ = v.RegisterValidation("userrole", ValidateUserRole)
	_ = v.RegisterValidation("strongpassword", ValidateStrongPassword)
	return v
}

// ─── ValidateSliceOfStruct ──────────────────────────────────────────────

func TestValidateSliceOfStruct_AllValid(t *testing.T) {
	v := newTestValidator()
	items := []testStruct{
		{Name: "Alice", Email: "alice@example.com", Age: 30},
		{Name: "Bob", Email: "bob@example.com", Age: 25},
	}
	err := ValidateSliceOfStruct(v, items)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}

func TestValidateSliceOfStruct_EmptySlice(t *testing.T) {
	v := newTestValidator()
	items := []testStruct{}
	err := ValidateSliceOfStruct(v, items)
	if err != nil {
		t.Errorf("empty slice should not error, got: %v", err)
	}
}

func TestValidateSliceOfStruct_MixedValidAndInvalid(t *testing.T) {
	v := newTestValidator()
	items := []testStruct{
		{Name: "Alice", Email: "alice@example.com", Age: 30},
		{Name: "", Email: "invalid", Age: 0},          // All fields invalid
		{Name: "Bob", Email: "bob@example.com", Age: 25},
		{Name: "X", Email: "not-email", Age: -5},      // Multiple invalid
	}
	err := ValidateSliceOfStruct(v, items)
	if err == nil {
		t.Fatal("expected error for mixed valid/invalid slice")
	}

	// Should return multiple errors (not just the last one).
	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	if len(validationErrs) < 2 {
		t.Errorf("expected at least 2 validation errors, got %d", len(validationErrs))
	}
}

func TestValidateSliceOfStruct_AllInvalid(t *testing.T) {
	v := newTestValidator()
	items := []testStruct{
		{Name: "", Email: "", Age: 0},
		{Name: "", Email: "", Age: 0},
	}
	err := ValidateSliceOfStruct(v, items)
	if err == nil {
		t.Fatal("expected error for all-invalid slice")
	}
	validationErrs, ok := err.(validator.ValidationErrors)
	if !ok {
		t.Fatalf("expected ValidationErrors, got %T", err)
	}
	// Each struct has 3 invalid fields → at least 6 errors.
	if len(validationErrs) < 6 {
		t.Errorf("expected at least 6 errors, got %d", len(validationErrs))
	}
}

// ─── ValidateUserRole ──────────────────────────────────────────────────

func TestValidateUserRole(t *testing.T) {
	tests := []struct {
		name  string
		role  string
		valid bool
	}{
		{"Admin valid", "Admin", true},
		{"Super Admin valid", "Super Admin", true},
		{"admin lowercase invalid", "admin", false},
		{"empty invalid", "", false},
		{"random string invalid", " Moderator", false},
		{"SuperAdmin no space", "SuperAdmin", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type roleHolder struct {
				Role string `validate:"required,userrole"`
			}
			v := newTestValidator()
			err := v.Struct(roleHolder{Role: tt.role})
			if tt.valid && err != nil {
				t.Errorf("role %q should be valid, got: %v", tt.role, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("role %q should be invalid", tt.role)
			}
		})
	}
}

// ─── ValidateStrongPassword ─────────────────────────────────────────────

func TestValidateStrongPassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		valid    bool
	}{
		{"valid", "Strongpass1!", true},
		{"valid complex", "MyP@ssw0rd#2024", true},
		{"too short", "Ab1!xy", false},
		{"no uppercase", "strongpass1!", false},
		{"no number", "Strongpass!", false},
		{"no symbol", "Strongpass1", false},
		{"empty", "", false},
		{"unicode", "Ströngpass1!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			type pwHolder struct {
				Password string `validate:"required,strongpassword"`
			}
			v := newTestValidator()
			err := v.Struct(pwHolder{Password: tt.password})
			if tt.valid && err != nil {
				t.Errorf("password %q should be valid, got: %v", tt.password, err)
			}
			if !tt.valid && err == nil {
				t.Errorf("password %q should be invalid", tt.password)
			}
		})
	}
}
