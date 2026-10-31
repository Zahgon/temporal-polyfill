package intl

import (
	"bytes"
	"compress/gzip"
	"embed"
	"encoding/json"
	"io"
	"math"
)

//go:embed data/*.gz
var dataFS embed.FS

func loadBytes(name string) []byte {
	f, err := dataFS.Open("data/" + name)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		panic(err)
	}
	var b bytes.Buffer
	if _, err := io.Copy(&b, zr); err != nil {
		panic(err)
	}
	return b.Bytes()
}

func loadJSON(name string, v any) {
	if err := json.Unmarshal(loadBytes(name+".json.gz"), v); err != nil {
		panic(err)
	}
}

func dataExists(name string) bool {
	f, err := dataFS.Open("data/" + name)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

func rangeError(msg string) *Error { return &Error{Name: "RangeError", Message: msg} }

func typeError(msg string) *Error { return &Error{Name: "TypeError", Message: msg} }

func floorDiv(a, b int64) int64 {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

func floorMod(a, b int64) int64 { return a - floorDiv(a, b)*b }

// validTime reports whether ms is a valid ECMAScript time value.
func validTime(ms float64) bool { return !math.IsNaN(ms) && math.Abs(ms) <= 8.64e15 }
