package gocrawler

import (
	"testing"
	//"net/http/httptest"
)

func assertErrors(t *testing.T, r *domainScan, e error, msg string) {

	if (r != nil) {
		t.Errorf("Wanted nil result " + msg)
	}

	if (e == nil) {
		t.Errorf("Wanted error " + msg)
	}

}

func TestErrorWhenBadUrl(t *testing.T)  {
	r,e := scan(":?::://www.example.com")

	assertErrors(t, r,e, "scan(badUrl)")

}

func TestErrorWhenNonHttp(t *testing.T) {
	r,e := scan("htt://www.example.com")

	assertErrors(t, r, e, "scan(nonHttpUrl")
}

func TestErrorWhenUnableToConnect(t *testing.T) {
	r,e := scan("http://localhost:162000")

	assertErrors(t, r,e, "scan(unableToConnect)")
}


func listContains(a string, list []string) int {
	for i,b := range list {
		if (b == a) {
			return i;
		}
	}
	return -1
}

func TestParseHrefs(t *testing.T) {
	html := `<!DOCTYPE html>
<html>
<body>

<h1>Test Document</h1>
<p><a href="/link1">My</a> first paragraph.</p>
<p><a _target="some target" class="something" href='/link2'>My</a> first paragraph.</p>
<p><a HREF='/link3'>My</a> first paragraph.</p>


</body>
</html>`

	links := findLinks(html)

	if (len(links) != 3) {
		t.Errorf("Found %d links, expecting 3", len(links))
	}

	for _, element := range []string{"/link1", "/link2", "/link3"}  {
		if (listContains(element, links) == -1) {
			t.Errorf("Found missing %s", element)
		}
	}

	if (listContains("/link2", links) == -1) {

	}
}

