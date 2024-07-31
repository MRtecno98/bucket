package bucket

func (oc *OpenContext) CleanCache() error {
	if oc.Database != nil {
		if err := oc.Database.Close(); err != nil {
			return err
		}
	}

	if err := oc.Fs.Remove(DATABASE_NAME); err != nil {
		return err
	}

	oc.Database = nil
	return nil
}
