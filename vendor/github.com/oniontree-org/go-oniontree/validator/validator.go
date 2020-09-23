package validator

import (
	"encoding/json"
	"errors"
	"github.com/xeipuuv/gojsonschema"
)

type Validator struct {
	schema string
}

func (v Validator) Validate(value interface{}) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}

	schemaLoader := gojsonschema.NewStringLoader(v.schema)
	documentLoader := gojsonschema.NewBytesLoader(b)

	res, err := gojsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return err
	}

	if !res.Valid() {
		errs := []error{}
		for _, errMsg := range res.Errors() {
			if errMsg == nil {
				continue
			}
			errs = append(errs, errors.New(errMsg.String()))
		}
		return &ValidatorError{errs}
	}

	return nil
}

func NewValidator(schema string) *Validator {
	return &Validator{
		schema: schema,
	}
}
