package app

import (
	"strings"
	"testing"

	"github.com/tinywasm/context"
	"github.com/tinywasm/devtui"
	"github.com/tinywasm/wizard"
)

type MockModule struct {
	steps []*wizard.Step
}

func (m *MockModule) Name() string { return "Mock" }
func (m *MockModule) GetSteps() []any {
	res := make([]any, len(m.steps))
	for i, s := range m.steps {
		res[i] = s
	}
	return res
}

func TestWizardLogsIntegration(t *testing.T) {
	// 1. Setup TUI with wizard tab
	tui := devtui.NewTUI(&devtui.TuiConfig{
		AppName: "WizardTest",
	})
	sectionWizard := tui.NewTabSection("WIZARD", "Project Initialization")

	// 2. Create real wizard steps
	steps := []*wizard.Step{
		{
			LabelText: "Project Name",
			DefaultFn: func(ctx *context.Context) string { return "" },
			OnInputFn: func(in string, ctx *context.Context) (bool, error) {
				if in == "" {
					return false, nil
				}
				_ = ctx.Set("name", in)
				return true, nil
			},
		},
		{
			LabelText: "Project Location",
			DefaultFn: func(ctx *context.Context) string { return "./" + ctx.Value("name") },
			OnInputFn: func(in string, ctx *context.Context) (bool, error) {
				return true, nil
			},
		},
	}

	// 3. Initialize Wizard
	var completed bool
	mockModule := &MockModule{steps: steps}
	w := wizard.New(func(ctx *context.Context) {
		completed = true
	}, mockModule)

	// 4. Register with TUI
	tui.AddHandler(w, 0, "#00ADD8", sectionWizard)

	// 5. Simulate Step 1 completion
	w.Change("myapp")

	// 6. Simulate Step 2 completion
	w.Change("./myapp")

	// 7. Verify states
	if !completed {
		t.Error("Wizard should be completed")
	}

	// 8. Verify Logs are preserved and correctly formatted
	// Note: index 1 because NewTUI adds SHORTCUTS as tab 0
	content := tui.ContentViewPlain(1)

	expectedLogs := []string{
		"✓ Project Name: myapp",
		"✓ Project Location: ./myapp",
	}

	for _, expected := range expectedLogs {
		if !strings.Contains(content, expected) {
			t.Errorf("Expected log %q not found in content:\n%s", expected, content)
		}
	}

	t.Logf("Final Wizard Logs:\n%s", content)
}
