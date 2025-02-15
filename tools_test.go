package godev

import "testing"

func TestGetModuleName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
		wantErr  bool
	}{
		{
			name:     "valid module path",
			path:     "miproject/modules/user/model.go",
			expected: "user",
			wantErr:  false,
		},
		{
			name:     "valid module path \\",
			path:     "miproject\\modules\\user\\model.go",
			expected: "user",
			wantErr:  false,
		},
		{
			name:     "valid module with multiple levels",
			path:     "app/src/modules/auth/login.go",
			expected: "auth",
			wantErr:  false,
		},
		{
			name:     "invalid path without modules",
			path:     "miproject/web/module3.wasm",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "empty path",
			path:     "",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "path ends in modules",
			path:     "project/modules",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetModuleName(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetModuleName(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Fatalf("GetModuleName(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

func TestGetFileName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
		wantErr  bool
	}{
		{
			name:     "valid path",
			path:     "theme/index.html",
			expected: "index.html",
			wantErr:  false,
		},
		{
			name:     "valid path backslash",
			path:     "theme\\index.html",
			expected: "index.html",
			wantErr:  false,
		},
		{
			name:     "only filename",
			path:     "index.html",
			expected: "index.html",
			wantErr:  false,
		},
		{
			name:     "empty path",
			path:     "",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "path ends in separator",
			path:     "theme/",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "path is separator",
			path:     "/",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "path is current dir",
			path:     ".",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetFileName(tt.path)
			if (err != nil) != tt.wantErr {
				t.Fatalf("GetFileName(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Fatalf("GetFileName(%q) = %v, want %v", tt.path, got, tt.expected)
			}
		})
	}
}

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
