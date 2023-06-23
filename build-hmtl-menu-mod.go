package godev

func (u ui) buildMenu() (public_menu, private_menu string) {

	var public_index int
	var private_index int
	for _, m := range u.modules {

		contains_public, contains_private := m.ContainsTypeAreas()

		if contains_public {

			public_menu += m.BuildMenuButton(public_index) + "\n"
			public_index++
		} else if contains_private {

			private_menu += m.BuildMenuButton(private_index) + "\n"
			private_index++
		}

	}

	return
}

func (u ui) buildModules() (public_modules, private_modules string) {

	for _, m := range u.modules {

		contains_public, contains_private := m.ContainsTypeAreas()

		if contains_public {

			public_modules += m.BuildHtmlModule() + "\n"

		} else if contains_private {

			private_modules += m.BuildHtmlModule() + "\n"
		}

	}

	return
}
