package temporal

// rejectNonStringPrimitive is RejectNonStringPrimitive of lib/primitive.ts.
func rejectNonStringPrimitive(input any) {
	if !isObject(input) {
		throwType("month code must be a string")
	}
}

// toPrimitiveRequireString is an inlined copy of ES.ToPrimitive with preferredType = String and only
// accepting String output. Objects in this package have no Symbol.toPrimitive, and their toString
// always answers with a string.
func toPrimitiveRequireString(inputParam any) string {
	input := unwrap(inputParam)
	if s, ok := input.(string); ok {
		return s
	}
	rejectNonStringPrimitive(input)
	result, ok := toPrimitive(input, "string").(string)
	if !ok {
		throwType("month code must be a string")
	}
	return result
}

// parseMonthCode is ParseMonthCode. A code holds only ASCII when it is valid, so a string with any
// other character fails the same test that the original's UTF-16 indexing fails.
func parseMonthCode(argument any) (monthNumber float64, isLeapMonth bool) {
	value := toPrimitiveRequireString(argument)
	valid := true
	for i := 0; i < len(value); i++ {
		if value[i] >= 0x80 {
			valid = false
		}
	}
	if valid {
		isLeapMonth = len(value) == 4
		switch {
		case len(value) < 3 || value[0] != 'M':
			valid = false
		case value[1] < '0' || value[1] > '9' || value[2] < '0' || value[2] > '9':
			valid = false
		default:
			monthNumber = float64(int(value[1]-'0')*10 + int(value[2]-'0'))
			if isLeapMonth {
				valid = value[3] == 'L'
			} else {
				valid = len(value) == 3 && monthNumber != 0
			}
		}
	}
	if !valid {
		throwRange("bad month code " + value + "; must match M01-M99 or M00L-M99L")
	}
	return monthNumber, isLeapMonth
}

// createMonthCode is CreateMonthCode.
func createMonthCode(monthNumber float64, isLeapMonth bool) string {
	numberPart := padStartZeros(numberToString(monthNumber), 2)
	if isLeapMonth {
		return "M" + numberPart + "L"
	}
	return "M" + numberPart
}
