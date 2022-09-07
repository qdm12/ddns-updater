package netcup

type Request struct {
	Action string `json:"action"`
	Param  Params `json:"param"`
}

func NewNetcupRequest(action string, params *Params) *Request {
	return &Request{
		Action: action,
		Param:  *params,
	}
}

type Params map[string]interface{}

func NewParams() Params {
	return make(map[string]interface{})
}

func (p Params) AddParam(key string, value interface{}) {
	p[key] = value
}
