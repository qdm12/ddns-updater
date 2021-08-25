package params

import "github.com/qdm12/golibs/params"

type envInterface interface {
	Get(key string, options ...params.OptionSetter) (value string, err error)
}
