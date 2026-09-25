package ariutil

import "testing"

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name: "valid config",
			cfg: Config{
				Application:  "myapp",
				Username:     "user",
				Password:     "pass",
				URL:          "http://localhost:8088/ari",
				WebsocketURL: "ws://localhost:8088/ari/events",
			},
			wantErr: false,
		},
		{name: "empty config", cfg: Config{}, wantErr: true},
		{
			name: "missing URL only",
			cfg: Config{
				Application:  "myapp",
				Username:     "user",
				Password:     "pass",
				WebsocketURL: "ws://localhost:8088/ari/events",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
