package godev

import "testing"

func TestIsFileType(t *testing.T) {
	tests := []struct {
		name         string
		filename     string
		wantFrontend bool
		wantBackend  bool
	}{
		{
			name:         "frontend file",
			filename:     "f.index.html",
			wantFrontend: true,
			wantBackend:  false,
		},
		{
			name:         "backend file",
			filename:     "b.main.go",
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
		{
			name:         "only prefix",
			filename:     "f.",
			wantFrontend: false,
			wantBackend:  false,
		},
		{
			name:         "case sensitive prefix",
			filename:     "F.index.html",
			wantFrontend: false,
			wantBackend:  false,
		},
		{
			name:         "wrong separator",
			filename:     "f-index.html",
			wantFrontend: false,
			wantBackend:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotFrontend, gotBackend := IsFileType(tt.filename)
			if gotFrontend != tt.wantFrontend || gotBackend != tt.wantBackend {
				t.Errorf("IsFileType(%q) = (%v, %v), want (%v, %v)",
					tt.filename, gotFrontend, gotBackend, tt.wantFrontend, tt.wantBackend)
			}
		})
	}
}
