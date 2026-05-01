package main

import "testing"

func TestParseAdminSpecsSupportsSingleAdmin(t *testing.T) {
	specs, err := parseAdminSpecs([]string{
		"ADMIN_EMAIL=3174285493@qq.com",
		"ADMIN_PASSWORD=secret-password",
	})
	if err != nil {
		t.Fatalf("expected single admin spec to parse, got %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("expected 1 admin spec, got %d", len(specs))
	}
	if specs[0].Email != "3174285493@qq.com" {
		t.Fatalf("expected email from env, got %q", specs[0].Email)
	}
	if specs[0].Username != "3174285493" {
		t.Fatalf("expected username derived from email, got %q", specs[0].Username)
	}
	if specs[0].Password != "secret-password" {
		t.Fatalf("expected password from env, got %q", specs[0].Password)
	}
}

func TestParseAdminSpecsSupportsMultipleAdmins(t *testing.T) {
	specs, err := parseAdminSpecs([]string{
		"ADMIN_EMAIL_2=second@example.com",
		"ADMIN_PASSWORD_2=second-password",
		"ADMIN_USERNAME_2=second-admin",
		"ADMIN_EMAIL_1=first@example.com",
		"ADMIN_PASSWORD_1=first-password",
		"ADMIN_USERNAME_1=first-admin",
	})
	if err != nil {
		t.Fatalf("expected multiple admin specs to parse, got %v", err)
	}
	if len(specs) != 2 {
		t.Fatalf("expected 2 admin specs, got %d", len(specs))
	}
	if specs[0].Email != "first@example.com" || specs[0].Username != "first-admin" {
		t.Fatalf("expected first indexed admin first, got %+v", specs[0])
	}
	if specs[1].Email != "second@example.com" || specs[1].Username != "second-admin" {
		t.Fatalf("expected second indexed admin second, got %+v", specs[1])
	}
}

func TestParseAdminSpecsRejectsMissingPassword(t *testing.T) {
	_, err := parseAdminSpecs([]string{
		"ADMIN_EMAIL_1=first@example.com",
	})
	if err == nil {
		t.Fatal("expected missing password to fail")
	}
}

func TestParseAdminSpecsRejectsEmptyConfiguration(t *testing.T) {
	_, err := parseAdminSpecs(nil)
	if err == nil {
		t.Fatal("expected empty admin configuration to fail")
	}
}
