package config

import (
	"time"

	zchainErrors "github.com/0chain/gosdk/errors"
)

type Directive map[string]interface{}

type Flow []Directive

func (d Directive) unwrap() (name string, val interface{}, ok bool) {
	for name, val = range d {
		ok = true
		return
	}
	return
}

func getNodeNames(val interface{}) (ss []NodeName, ok bool) {
	switch tt := val.(type) {
	case string:
		return []NodeName{NodeName(tt)}, true
	case []interface{}:
		ss = make([]NodeName, 0, len(tt))
		for _, t := range tt {
			if ts, ok := t.(string); ok {
				ss = append(ss, NodeName(ts))
			} else {
				return nil, false
			}
		}
		return ss, true
	case []string:
		ss = make([]NodeName, 0, len(tt))
		for _, t := range tt {
			ss = append(ss, NodeName(t))
		}
		return ss, true
	}
	return // nil, false
}

func (d Directive) Execute(ex Executor) (err error, mustFail bool) {
	var tm = 10 * time.Minute // default timeout is 10 minutes

	var name, val, ok = d.unwrap()
	if !ok {
		return zchainErrors.New("invalid empty flow"), false
	}

	var mf bool = false
	if msi, ok := val.(map[interface{}]interface{}); ok {
		// extract timeout
		if tmsi, ok := msi["timeout"]; ok {
			tms, ok := tmsi.(string)
			if !ok {
				return zchainErrors.Newf("", "invalid 'timeout' type: %T", tmsi), false
			}
			if tm, err = time.ParseDuration(tms); err != nil {
				return zchainErrors.Newf("", "paring 'timeout' %q: %v", tms, err), false
			}
			delete(msi, "timeout")
		}

		// extract must_fail
		if mfmsi, ok := msi["must_fail"]; ok {
			mf, ok = mfmsi.(bool)
			if !ok {
				return zchainErrors.Newf("", "invalid 'must_fail' type: %T", mfmsi), false
			}
			delete(msi, "must_fail")
		}
	}

	err = execute(name, ex, val, tm)
	return err, mf
}

func execute(name string, ex Executor, val interface{}, tm time.Duration) (
	err error) {

	var fn, ok = flowRegistry[name]
	if !ok {
		return zchainErrors.Newf("", "unknown flow directive: %q", name)
	}

	return fn(name, ex, val, tm)
}
