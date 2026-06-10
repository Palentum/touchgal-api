package repository

import "testing"

var benchmarkLikePattern string

func BenchmarkLikeContainsPattern(b *testing.B) {
	cases := []struct {
		name  string
		value string
	}{
		{name: "Plain", value: "summer game"},
		{name: "Escaped", value: `summer_%\\game`},
	}
	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				benchmarkLikePattern = likeContainsPattern(tc.value)
			}
		})
	}
}
