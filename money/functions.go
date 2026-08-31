// money - exact money types for Go
//
// Copyright (c) 2026 Michael D Henderson
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package money

// Abs returns the absolute value of the amount.
func Abs(m Money) Money {
	return m.Abs()
}

// Add returns the sum of same-currency money values.
func Add(m Money, values ...Money) (Money, error) {
	return m.Add(values...)
}

// Subtract returns the difference of same-currency money values.
func Subtract(m Money, values ...Money) (Money, error) {
	return m.Subtract(values...)
}

// Compare returns -1, 0, or 1 for same-currency money values.
func Compare(m, other Money) (int, error) {
	return m.Compare(other)
}

// Equals reports whether two same-currency money values have the same amount.
func Equals(m, other Money) (bool, error) {
	return m.Equals(other)
}

// IsGreaterThan reports whether m is greater than other.
func IsGreaterThan(m, other Money) (bool, error) {
	return m.IsGreaterThan(other)
}

// IsLessThan reports whether m is less than other.
func IsLessThan(m, other Money) (bool, error) {
	return m.IsLessThan(other)
}

// IsNegative reports whether the amount is less than zero.
func IsNegative(m Money) bool {
	return m.IsNegative()
}

// IsPositive reports whether the amount is greater than zero.
func IsPositive(m Money) bool {
	return m.IsPositive()
}

// IsZero reports whether the amount is zero.
func IsZero(m Money) bool {
	return m.IsZero()
}
