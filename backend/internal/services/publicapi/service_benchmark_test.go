package publicapi

import "testing"

func BenchmarkNormalizeSearch(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		keyword, page, limit, err := NormalizeSearch(" summer ", 0, 999)
		if err != nil || keyword != "summer" || page != 1 || limit != maxSearchLimit {
			b.Fatalf("unexpected normalization: %q %d %d %v", keyword, page, limit, err)
		}
	}
}

func BenchmarkNormalizeSearchUnicode(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		keyword, page, limit, err := NormalizeSearch("サマーゲーム", 3, 25)
		if err != nil || keyword != "サマーゲーム" || page != 3 || limit != 25 {
			b.Fatalf("unexpected unicode normalization: %q %d %d %v", keyword, page, limit, err)
		}
	}
}
