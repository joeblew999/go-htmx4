package live_test

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/joeblew999/go-htmx4/kit/live"
)

// ValidTopic and TopicPattern must agree: Go checks one, the Worker entry's JavaScript the other.
func TestValidTopicMatchesPattern(t *testing.T) {
	re := regexp.MustCompile(live.TopicPattern)
	cases := []string{"", "a", "lobby", "t-1", "-", "--", "UPPER", "a_b", "a.b", "a b", "ü", "日本",
		strings.Repeat("a", 32), strings.Repeat("a", 33), "a\n", "\na", "0123456789", "a/b", "a%20b"}
	for c := range 128 {
		cases = append(cases, "x"+string(rune(c)))
	}
	for _, c := range cases {
		if got, want := live.ValidTopic(c), re.MatchString(c); got != want {
			t.Errorf("ValidTopic(%q) = %v, TopicPattern says %v", c, got, want)
		}
	}
}

func ExampleValidTopic() {
	for _, t := range []string{"lobby", "team-42", "Lobby", "a/b"} {
		fmt.Println(t, live.ValidTopic(t))
	}
	// Output:
	// lobby true
	// team-42 true
	// Lobby false
	// a/b false
}
