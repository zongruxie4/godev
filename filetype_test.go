package godev

import "testing"

func TestIsFileType(t *testing.T) {
	ft := GoFileType{
		FrontendPrefix: []string{"f.", "front."},
		FrontendFiles:  []string{"wasm.main.go"},
		BackendPrefix:  []string{"b.", "back."},
		BackendFiles:   []string{"main.server.go"},
	}

	tests := []struct {
		name         string
		filename     string
		wantFrontend bool
		wantBackend  bool
	}{
		{
			name:         "frontend prefix f.",
			filename:     "f.index.html",
			wantFrontend: true,
			wantBackend:  false,
		},
		{
			name:         "frontend prefix front.",
			filename:     "front.index.html",
			wantFrontend: true,
			wantBackend:  false,
		},
		{
			name:         "frontend specific file",
			filename:     "wasm.main.go",
			wantFrontend: true,
			wantBackend:  false,
		},
		{
			name:         "backend prefix b.",
			filename:     "b.main.go",
			wantFrontend: false,
			wantBackend:  true,
		},
		{
			name:         "backend prefix back.",
			filename:     "back.main.go",
			wantFrontend: false,
			wantBackend:  true,
		},
		{
			name:         "backend specific file",
			filename:     "main.server.go",
			wantFrontend: false,
			wantBackend:  true,
		},
		{
			name:         "regular file",
			filename:     "index.html",
			wantFrontend: false,
			wantBackend:  false,
		},
		{
			name:         "empty string",
			filename:     "",
			wantFrontend: false,
			wantBackend:  false,
		},
		{
			name:         "single character",
			filename:     "f",
			wantFrontend: false,
			wantBackend:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFrontend, gotBackend := ft.GoFileIsType(tt.filename)
			if gotFrontend != tt.wantFrontend || gotBackend != tt.wantBackend {
				t.Errorf("GoFileIsType(%q) = (%v, %v), want (%v, %v)",
					tt.filename, gotFrontend, gotBackend, tt.wantFrontend, tt.wantBackend)
			}
		})
	}
}
