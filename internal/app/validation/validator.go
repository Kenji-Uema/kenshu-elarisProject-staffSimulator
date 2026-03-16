package validation

type Validator struct {
	steps []func() error
}

func New() *Validator {
	return &Validator{
		steps: make([]func() error, 0),
	}
}

func (v *Validator) Validate() error {
	for _, step := range v.steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}
