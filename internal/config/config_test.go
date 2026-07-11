package config

import "testing"

func TestAgentConfigValidateProduction(t *testing.T) {
	tests := []struct {
		name    string
		cfg     AgentConfig
		wantErr bool
	}{
		{
			name: "dev mode allows http",
			cfg: AgentConfig{
				APIURL:      "http://127.0.0.1:8080",
				EnrollToken: "",
			},
		},
		{
			name: "production requires https",
			cfg: AgentConfig{
				Production:  true,
				APIURL:      "http://127.0.0.1:8080",
				EnrollToken: "secret",
			},
			wantErr: true,
		},
		{
			name: "production requires enroll token",
			cfg: AgentConfig{
				Production: true,
				APIURL:     "https://api.example.com",
			},
			wantErr: true,
		},
		{
			name: "production ok",
			cfg: AgentConfig{
				Production:  true,
				APIURL:      "https://api.example.com",
				EnrollToken: "secret",
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.wantErr && err == nil {
				t.Fatal("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestAPIConfigValidateProduction(t *testing.T) {
	if err := (APIConfig{Production: true}).Validate(); err == nil {
		t.Fatal("expected error without enroll token")
	}
	if err := (APIConfig{Production: true, EnrollToken: "secret"}).Validate(); err == nil {
		t.Fatal("expected error without redis")
	}
	if err := (APIConfig{
		Production:  true,
		EnrollToken: "secret",
		RedisURL:    "redis://127.0.0.1:6379/0",
	}).Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAPIConfigPersistFiles(t *testing.T) {
	dev := APIConfig{Production: false}
	if !dev.PersistFiles() {
		t.Fatal("dev should persist files by default")
	}
	prod := APIConfig{Production: true}
	if prod.PersistFiles() {
		t.Fatal("production should not persist files by default")
	}
}
