// Package temporal is a Go port of @js-temporal/polyfill, the TC39 Temporal polyfill. It keeps the
// original's behaviour byte for byte: every output and every exception (constructor name and
// message) equals what the TypeScript original produces.
//
// JavaScript semantics carry over directly. Every Number is a float64 and every BigInt a *big.Int;
// a nil argument stands for undefined and Null for null. Bags and option structs have *Value
// fields, so an absent property is a nil pointer. Exceptions are raised as panic(*Error).
package temporal

// oneProp is the shape that round(), total() and getTimeZoneTransition() share: a string argument
// stands for the one named option, anything else must be an options object.
func oneProp(key string, param any) any {
	if jsTypeof(param) == "string" {
		return createOnePropObject(key, unwrap(param))
	}
	return getOptionsObject(param)
}

// defaultTo is a default parameter: an undefined value becomes fallback.
func defaultTo(value, fallback any) any {
	if isUndefined(value) {
		return fallback
	}
	return unwrap(value)
}

func stringOrUndefined(p *string) any {
	if p == nil {
		return nil
	}
	return *p
}

func numberOrUndefined(p *float64) any {
	if p == nil {
		return nil
	}
	return *p
}

// spreadTimeIntoFields is the ...isoDateTime.time of an object spread into a fields record.
func spreadTimeIntoFields(fields *calendarFields, time timeRecord) {
	fields.hour = &time.hour
	fields.minute = &time.minute
	fields.second = &time.second
	fields.millisecond = &time.millisecond
	fields.microsecond = &time.microsecond
	fields.nanosecond = &time.nanosecond
}
