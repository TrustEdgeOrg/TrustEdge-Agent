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
