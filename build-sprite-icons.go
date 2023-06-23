package godev

func (u ui) buildSpriteIcons() (public_icons, private_icons string) {

	for _, m := range u.modules {

		contains_public, contains_private := m.ContainsTypeAreas()

		if contains_public {

			public_icons += m.BuildIconModule()

		} else if contains_private {

			private_icons += m.BuildIconModule()
		}
	}

	return
}
