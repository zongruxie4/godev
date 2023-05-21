package godev

func init() {
	checkFolders()
	copyStaticFiles()
}

func (u ui) buildAll() {

	u.BuildCSS()
	u.BuildHTML()
	u.BuildJS()
	u.BuildWASM()

}
