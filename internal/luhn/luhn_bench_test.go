package luhn

import "testing"

// BenchmarkValid measures the throughput of Valid against a representative
// 16-digit number that passes the Luhn check.
func BenchmarkValid(b *testing.B) {
	const number = "4561261212345467"
	for b.Loop() {
		_ = Valid(number)
	}
}
