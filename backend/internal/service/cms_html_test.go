package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCMSPolicyKeepsContentMarkupAndStripsActiveContent(t *testing.T) {
	policy := NewCMSPolicy()
	cases := []struct {
		name string
		in   string
		want string
		not  []string
	}{
		{"headings lists tables", `<h2>T</h2><ul><li>a</li></ul><table><tbody><tr><td colspan="2">x</td></tr></tbody></table>`,
			`<h2>T</h2><ul><li>a</li></ul><table><tbody><tr><td colspan="2">x</td></tr></tbody></table>`, nil},
		{"https link", `<a href="https://example.com/a">x</a>`, `href="https://example.com/a"`, nil},
		{"relative link", `<a href="/career">x</a>`, `href="/career"`, nil},
		{"mailto and tel", `<a href="mailto:a@b.co">m</a><a href="tel:+62123456">t</a>`, `href="tel:+62123456"`, nil},
		{"http link dropped", `<a href="http://example.com">x</a>`, "", []string{"http://example.com"}},
		{"javascript link dropped", `<a href="javascript:alert(1)">x</a>`, "", []string{"javascript"}},
		{"protocol relative dropped", `<a href="//evil.com">x</a>`, "", []string{"evil.com"}},
		{"data link dropped", `<a href="data:text/html,<script>1</script>">x</a>`, "", []string{"data:"}},
		{"external links open safely", `<a href="https://example.com">x</a>`, `noopener`, nil},
		{"image http and https", `<img src="http://example.com/a.png" alt="a"><img src="https://example.com/b.png" alt="b">`, `src="https://example.com/b.png"`, nil},
		{"image data uri dropped", `<img src="data:image/png;base64,AAAA" alt="a">`, "", []string{"data:"}},
		{"scripts and handlers", `<p onclick="x()">a</p><script>alert(1)</script><iframe src="https://x"></iframe><style>p{}</style>`, `<p>a</p>`, []string{"script", "onclick", "iframe", "style"}},
		{"inline style limited", `<p style="text-align:center;position:fixed">a</p>`, `text-align: center`, []string{"position"}},
		{"unsafe class dropped", `<p class="a b">x</p><p class="x&quot;onload=">y</p>`, `class="a b"`, []string{"onload"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := policy.Sanitize(c.in)
			if c.want != "" {
				assert.Contains(t, got, c.want)
			}
			for _, n := range c.not {
				assert.NotContains(t, got, n)
			}
		})
	}
}
