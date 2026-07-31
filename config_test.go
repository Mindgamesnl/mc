package main

import (
	"os"
	"testing"
)

func TestLoadConfigNotExists(t *testing.T) {
	t.Chdir(t.TempDir())

	config, exists, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v, want nil", err)
	}
	if exists {
		t.Error("loadConfig() exists = true, want false")
	}
	if config != nil {
		t.Errorf("loadConfig() config = %v, want nil", config)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	t.Chdir(t.TempDir())

	want := &Config{Version: "26.2.1", Memory: "4G", Port: 25566}
	if err := saveConfig(want); err != nil {
		t.Fatalf("saveConfig() error = %v", err)
	}

	got, exists, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig() error = %v", err)
	}
	if !exists {
		t.Fatal("loadConfig() exists = false, want true")
	}
	if *got != *want {
		t.Errorf("loadConfig() = %+v, want %+v", *got, *want)
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	t.Chdir(t.TempDir())

	if err := os.WriteFile(configFile, []byte("version: [unterminated"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := loadConfig(); err == nil {
		t.Error("loadConfig() error = nil, want a parse error")
	}
}

func TestConfigNormalize(t *testing.T) {
	tests := []struct {
		name    string
		in      Config
		want    Config
		wantErr bool
	}{
		{
			name: "fills defaults",
			in:   Config{Version: "26.2"},
			want: Config{Version: "26.2", Memory: defaultMemory, Port: defaultPort},
		},
		{
			name: "keeps explicit values",
			in:   Config{Version: "1.21.4", Memory: "512M", Port: 25570},
			want: Config{Version: "1.21.4", Memory: "512M", Port: 25570},
		},
		{name: "missing version", in: Config{Memory: "2G"}, wantErr: true},
		{name: "invalid version", in: Config{Version: "banana"}, wantErr: true},
		{name: "invalid memory", in: Config{Version: "26.2", Memory: "lots"}, wantErr: true},
		{name: "port too high", in: Config{Version: "26.2", Port: 70000}, wantErr: true},
		{name: "negative port", in: Config{Version: "26.2", Port: -1}, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := test.in
			err := config.normalize()
			if test.wantErr {
				if err == nil {
					t.Fatalf("normalize() error = nil, want an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("normalize() error = %v", err)
			}
			if config != test.want {
				t.Errorf("normalize() = %+v, want %+v", config, test.want)
			}
		})
	}
}
