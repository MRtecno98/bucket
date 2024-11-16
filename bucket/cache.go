package bucket

func (cp *CachedPlugin) GetName() string {
	if cp.RemotePlugin != nil {
		return cp.RemotePlugin.GetName()
	}

	return cp.name
}

func (cp *CachedPlugin) GetIdentifier() string {
	if cp.RemotePlugin != nil {
		return cp.RemotePlugin.GetIdentifier()
	}

	return cp.RemoteIdentifier
}

func (cp *CachedPlugin) GetAuthors() []string {
	if cp.RemotePlugin != nil {
		return cp.RemotePlugin.GetAuthors()
	}

	return cp.authors
}

func (cp *CachedPlugin) GetDescription() string {
	if cp.RemotePlugin != nil {
		return cp.RemotePlugin.GetDescription()
	}

	return cp.description
}

func (cp *CachedPlugin) GetWebsite() string {
	if cp.RemotePlugin != nil {
		return cp.RemotePlugin.GetWebsite()
	}

	return cp.website
}

func (cp *CachedPlugin) requestIfMissing() error {
	if cp.RemotePlugin == nil {
		return cp.Request()
	} else {
		return nil
	}
}

func (cp *CachedPlugin) GetLatestVersion() (RemoteVersion, error) {
	if err := cp.requestIfMissing(); err != nil {
		return nil, err
	}

	return cp.RemotePlugin.GetLatestVersion()
}

func (cp *CachedPlugin) GetVersions() ([]RemoteVersion, error) {
	if err := cp.requestIfMissing(); err != nil {
		return nil, err
	}

	return cp.RemotePlugin.GetVersions()
}

func (cp *CachedPlugin) GetVersion(version string) (RemoteVersion, error) {
	if err := cp.requestIfMissing(); err != nil {
		return nil, err
	}

	return cp.RemotePlugin.GetVersion(version)
}

func (cp *CachedPlugin) GetVersionIdentifiers() ([]string, error) {
	if err := cp.requestIfMissing(); err != nil {
		return nil, err
	}

	return cp.RemotePlugin.GetVersionIdentifiers()
}

func (cp *CachedPlugin) GetLatestCompatible(plt PlatformType) (RemoteVersion, error) {
	if err := cp.requestIfMissing(); err != nil {
		return nil, err
	}

	return cp.RemotePlugin.GetLatestCompatible(plt)
}

func (cp *CachedPlugin) GetRepository() Repository {
	if cp.RemotePlugin != nil {
		return cp.RemotePlugin.GetRepository()
	}

	return cp.Repository.Repository
}
