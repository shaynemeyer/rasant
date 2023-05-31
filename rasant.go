package rasant

const version = "1.0.0"

type Rasant struct {
	AppName string
	Debug bool
	Version string
}

func (ras *Rasant) New(rootPath string) error {
	pathConfig := initPaths{
		rootPath: rootPath,
		folderNames: []string{"handlers", "migrations", "views", "data", "public", "tmp", "logs", "middleware"},
	}

	err := ras.Init(pathConfig)
	if err!= nil {
		return err
	}

	return nil
}

func (ras *Rasant) Init(p initPaths) error {
	root := p.rootPath
	for _, path := range p.folderNames {
		// create directory if it doesn't exist
		err := ras.CreateDirIfNotExist(root + "/" + path)
		if err!= nil {
      return err
    }
	}
	return nil
}