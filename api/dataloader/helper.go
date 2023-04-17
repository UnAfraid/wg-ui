package dataloader

func repeatError(err error, count int) []error {
	var errs []error
	for i := 0; i < count; i++ {
		errs = append(errs, err)
	}
	return errs
}
