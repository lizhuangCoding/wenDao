package main

import "testing"

func TestParseAdminSpecsSupportsSingleAdmin(t *testing.T) {
	spec, err := parseAdminSpec([]string{
		"ADMIN_EMAIL=admin@example.com",
		"ADMIN_PASSWORD=secret-password",
	})
	if err != nil {
		t.Fatalf("expected single admin spec to parse, got %v", err)
	}
	if spec.Email != "admin@example.com" {
		t.Fatalf("expected email from env, got %q", spec.Email)
	}
	if spec.Username != "admin" {
		t.Fatalf("expected username derived from email, got %q", spec.Username)
	}
	if spec.Password != "secret-password" {
		t.Fatalf("expected password from env, got %q", spec.Password)
	}
}

func TestParseAdminSpecRejectsNumberedAdminVariables(t *testing.T) {
	_, err := parseAdminSpec([]string{
		"ADMIN_EMAIL=admin@example.com",
		"ADMIN_PASSWORD=secret-password",
		"ADMIN_EMAIL_1=other@example.com",
		"ADMIN_PASSWORD_1=other-password",
	})
	if err == nil {
		t.Fatal("expected numbered admin variables to fail")
	}
}

func TestParseAdminSpecRejectsMissingPassword(t *testing.T) {
	_, err := parseAdminSpec([]string{
		"ADMIN_EMAIL=admin@example.com",
	})
	if err == nil {
		t.Fatal("expected missing password to fail")
	}
}

func TestParseAdminSpecRejectsEmptyConfiguration(t *testing.T) {
	_, err := parseAdminSpec(nil)
	if err == nil {
		t.Fatal("expected empty admin configuration to fail")
	}
}
