package config

import (
	"errors"
	"fmt"
	"time"
)

// The Flow represents single value map.
type Flow map[string]interface{}

func (f Flow) getFirst() (name string, val interface{}, ok bool) {
	for name, val = range f {
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

func (f Flow) execute(name string, ex Executor, val interface{},
	tm time.Duration) (err error) {

	var fn, ok = flowRegistry[name]
	if !ok {
		return fmt.Errorf("unknown flow directive: %q", name)
	}

	return fn(f, name, ex, val, tm)
}

// Execute the flow directive.
func (f Flow) Execute(ex Executor) (err error) {
	var name, val, ok = f.getFirst()
	if !ok {
		return errors.New("invalid empty flow")
	}

	var tm time.Duration

	// extract timeout
	if msi, ok := val.(map[interface{}]interface{}); ok {
		if tmsi, ok := msi["timeout"]; ok {
			tms, ok := tmsi.(string)
			if !ok {
				return fmt.Errorf("invalid 'timeout' type: %T", tmsi)
			}
			if tm, err = time.ParseDuration(tms); err != nil {
				return fmt.Errorf("paring 'timeout' %q: %v", tms, err)
			}
			delete(msi, "timeout")
		}
	}

	return f.execute(name, ex, val, tm)
}

// Flows represents order of start/stop miners/sharder and other BC events.
type Flows []Flow
