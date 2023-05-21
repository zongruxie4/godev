package theme

func (Platform) ModuleJsTemplate() string {
	return `MODULES['%v'] = (function () {
		let crud = new Object();
		const module = document.getElementById('%v');
		%v
		crud.ListenerModuleON = function () {
		 %v
		};

		crud.ListenerModuleOFF = function () {
		 %v
		};
		return crud;
	})();`
}
