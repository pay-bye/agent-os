package config

func Load(input Input) (Values, error) {
	file, err := readConfigFile(input.File)
	if err != nil {
		return Values{}, err
	}
	config := Values{
		DatabaseURL: choose(input.DatabaseURL, file.DatabaseURL, envDatabaseURL(input.Env)),
		Listen:      choose(input.Listen, file.Listen),
		Declaration: choose(input.Declaration, file.Declaration, DefaultDeclaration),
		Grace:       chooseDuration(input.Grace, seconds(file.GraceSeconds), defaultGrace()),
	}
	if err := validate(config, input); err != nil {
		return Values{}, err
	}
	return config, nil
}
