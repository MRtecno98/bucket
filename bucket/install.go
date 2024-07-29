package bucket

func (c *OpenContext) InstallLatest(plugin RemotePlugin) error {
	latest, err := plugin.GetLatestVersion()
	if err != nil {
		return err
	}

	return c.InstallVersion(latest)
}

func (c *OpenContext) InstallVersion(ver RemoteVersion) error {
	return nil
}
