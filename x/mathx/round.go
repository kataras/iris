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

// RoundToInteger rounds the given float64 to an integer.
func RoundToInteger(x float64) int {
	// If the number is 1.1 round it to the previous integer (1), if >= 1.11 round it to the next one (2).
	t := math.Trunc(x)
	odd := math.Remainder(t, 2) != 0
	d := math.Abs(x - t)
	d = Round(d, 0.5, 2) // round to 2 decimals so we can easily check 0.11 and 0.1 and 0.2.
	if d > 0.1 || (d == 0.2 && odd) {
		// fmt.Printf("%f-%f -> %f. Is > 0.1 -> %v \n", x, t, d, d > 0.1)
		t = t + math.Copysign(1, x)
		return int(t)
	}
	return int(t)
}
