package theme

type Platform struct{}

func (Platform) PathTemplateIndexHTML() string {
	return "ui/theme/layout/platform.html"
}
