package main

import (
	"net/http/httptest"
	"testing"
)

// BenchmarkPages renders each page as a request would, with standard Go: time and, above all, memory per
// page. Memory is what costs on Workers, where TinyGo's allocator and collector are far slower:
//
//	go test -run '^$' -bench Pages -benchmem
func BenchmarkPages(b *testing.B) {
	h := newServer().routes()
	for _, p := range []string{"/de/", "/board", "/de/formats", "/de/about"} {
		b.Run(p, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				w := httptest.NewRecorder()
				h.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
				if w.Code != 200 {
					b.Fatalf("GET %s: %d", p, w.Code)
				}
			}
		})
	}
}
