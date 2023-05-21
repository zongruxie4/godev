package theme

func (Platform) MenuButtonTemplate() string {
	return `<li class="navbar-item"><a href="#%v" tabindex="%v" class="navbar-link" name="%v">
	<svg aria-hidden="true" focusable="false" class="fa-primary"><use xlink:href="#%v" /></svg>
	<span class="link-text">%v</span></a></li>`
}

func (Platform) ModuleHtmlTemplate() string {
	// nombre del module html y el contenido
	return `<div id="%v" class="slider_panel">%v</div>`
}
