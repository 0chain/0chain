package common

/*Lookup - used to track a code, value pair. code is used in the program and value is user friendly label */
type Lookup struct {
	Code  string `json:"code"`
	Value string `json:"value"`
}

/*GetCode - get the code */
func (l *Lookup) GetCode() string {
	return l.Code
}

/*GetValue - get the value */
func (l *Lookup) GetValue() string {
	return l.Value
}

/*CreateLookups - given a code,value args, return an array of lookups */
func CreateLookups(arg ...string) []*Lookup {
	lookups := make([]*Lookup, 0, len(arg)/2)
	for i := 0; i < len(arg); i += 2 {
		lookups = append(lookups, &Lookup{Code: arg[i], Value: arg[i+1]})
	}
	return lookups
}
