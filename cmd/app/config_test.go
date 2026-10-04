package main

import "testing"

func setConfigEnv(t *testing.T) {
	t.Helper()
	for key, value := range map[string]string{
		"PORT":                "",
		"DATABASE_URL":        "postgres://example/database",
		"POSTGRESQL_STR":      "",
		"REDIS_ADDR":          "",
		"REDIS_URL":           "",
		"TRAVELPAYOUTS_TOKEN": "test-token",
		"JWT_SECRET":          "test-secret",
		"ENABLE_PPROF":        "",
	} {
		t.Setenv(key, value)
	}
}

func TestLoadConfigDefaultsAndProfilerOptIn(t *testing.T) {
	setConfigEnv(t)
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.port != "8080" || cfg.redisAddr != "localhost:6379" || cfg.enableProfiler {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	t.Setenv("ENABLE_PPROF", "true")
	cfg, err = loadConfig()
	if err != nil || !cfg.enableProfiler {
		t.Fatalf("profiler opt-in was not applied: %v", err)
	}
}

func TestLoadConfigSupportsLegacyNames(t *testing.T) {
	setConfigEnv(t)
	t.Setenv("DATABASE_URL", "")
	t.Setenv("POSTGRESQL_STR", "postgres://legacy/database")
	t.Setenv("REDIS_URL", "legacy-redis:6379")
	cfg, err := loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.databaseURL != "postgres://legacy/database" || cfg.redisAddr != "legacy-redis:6379" {
		t.Fatal("legacy configuration names were ignored")
	}
	t.Setenv("DATABASE_URL", "postgres://current/database")
	t.Setenv("REDIS_ADDR", "current-redis:6379")
	cfg, err = loadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.databaseURL != "postgres://current/database" || cfg.redisAddr != "current-redis:6379" {
		t.Fatal("preferred configuration names did not take precedence")
	}
}

func TestLoadConfigRejectsInvalidConfiguration(t *testing.T) {
	for _, tc := range []struct{ name, key, value string }{
		{"zero port", "PORT", "0"},
		{"large port", "PORT", "65536"},
		{"non-numeric port", "PORT", "http"},
		{"missing database", "DATABASE_URL", ""},
		{"missing API token", "TRAVELPAYOUTS_TOKEN", ""},
		{"missing JWT secret", "JWT_SECRET", ""},
		{"invalid profiler flag", "ENABLE_PPROF", "maybe"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			setConfigEnv(t)
			t.Setenv(tc.key, tc.value)
			if _, err := loadConfig(); err == nil {
				t.Fatal("expected invalid configuration to be rejected")
			}
		})
	}
}
