package module_info

import (
	"path/filepath"
	"runtime"

	"github.com/cdvelop/model"
)

type module struct{}

func Add(theme model.Theme) *model.Module {

	m := model.Module{
		Theme:    theme,
		MainName: "module_info",
		Title:    "Información Plataforma TEST",
		Icon: model.Icon{
			Id:      "icon-info",
			ViewBox: "0 0 16 16",
			Paths:   []string{"m7 11h2v2h-2zm4-7c0.552 0 1 0.448 1 1v3l-3 2h-2v-1l3-2v-1h-5v-2h6zm-3-2.5c-1.736 0-3.369 0.676-4.596 1.904s-1.904 2.86-1.904 4.596 0.676 3.369 1.904 4.596 2.86 1.904 4.596 1.904 3.369-0.676 4.596-1.904 1.904-2.86 1.904-4.596-0.676-3.369-1.904-4.596-2.86-1.904-4.596-1.904zm0-1.5c4.418 0 8 3.582 8 8s-3.582 8-8 8-8-3.582-8-8 3.582-8 8-8z"},
		},
		Areas:   []byte{'0', 't'},
		Objects: []*model.Object{},
		Path:    module{},
	}

	return &m
}

func (module) FolderPath() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	return filepath.ToSlash(dir)
}
