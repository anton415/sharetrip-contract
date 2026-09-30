package config

import "testing"

func TestLoad(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://postgres:test-password@postgres/sharetrip_contract")
	t.Setenv("HTTP_ADDR", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":8080" || cfg.DatabaseURL != "postgres://postgres:test-password@postgres/sharetrip_contract" {
		t.Fatal("unexpected configuration")
	}
	t.Setenv("HTTP_ADDR", ":8082")
	cfg, err = Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != ":8082" {
		t.Fatal("HTTP_ADDR override was not applied")
	}
}

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	cfg, err := Load()
	if err == nil || err.Error() != "DATABASE_URL is required" || cfg != (Config{}) {
		t.Fatal("expected DATABASE_URL configuration error")
	}
}
