// Package probe checks a running fresh Worker at BASE: on local workerd (charter with) or deployed.
package probe

import (
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"testing"
)

func get(t *testing.T, method, url string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(method, url, nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	b, _ := io.ReadAll(res.Body)
	return res.StatusCode, string(b)
}

var count = regexp.MustCompile(`(\d+) visits`)

// TestProbe: the page renders with the environment's variable, and a visit adds one to D1. PROBE_LIMIT=1 also uses
// up the rate limit (3 a minute), which is exact on workerd and eventually consistent on Cloudflare.
func TestProbe(t *testing.T) {
	base := os.Getenv("BASE")
	if base == "" {
		t.Skip("BASE: a running fresh Worker")
	}
	status, page := get(t, "GET", base+"/")
	m := count.FindStringSubmatch(page)
	if status != 200 || m == nil || !regexp.MustCompile(`<p id="env">\S`).MatchString(page) {
		t.Fatalf("GET /: %d\n%s", status, page)
	}
	before, _ := strconv.Atoi(m[1])
	status, body := get(t, "POST", base+"/visit")
	if status != 200 || body != `<p id="visits">`+strconv.Itoa(before+1)+` visits</p>` {
		t.Errorf("POST /visit: %d %q, want %d visits", status, body, before+1)
	}
	if os.Getenv("PROBE_LIMIT") == "1" {
		get(t, "POST", base+"/visit")
		get(t, "POST", base+"/visit")
		if status, _ := get(t, "POST", base+"/visit"); status != 429 {
			t.Errorf("a 4th visit in a minute: %d, want 429", status)
		}
	}
}
