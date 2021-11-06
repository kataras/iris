package mathx

import "math"

// Round rounds the "input" on "roundOn" (e.g. 0.5) on "places" digits.
func Round(input float64, roundOn float64, places float64) float64 {
	pow := math.Pow(10, places)
	digit := pow * input

	_, div := math.Modf(digit)
	if div >= roundOn {
		return math.Ceil(digit) / pow
	}

	return math.Floor(digit) / pow
}

// RoundUp rounds up the "input" up to "places" digits.
func RoundUp(input float64, places float64) float64 {
	pow := math.Pow(10, places)
	return math.Ceil(pow*input) / pow
}

// RoundDown rounds down the "input" up to "places" digits.
func RoundDown(input float64, places float64) float64 {
	pow := math.Pow(10, places)
	return math.Floor(pow*input) / pow
}
