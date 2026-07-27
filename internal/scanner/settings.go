package scanner

func ValidateScheduledScanInterval(value *int64) error {
	if value != nil && (*value < 1 || *value > 8760) {
		return ErrInvalidEntry
	}
	return nil
}
