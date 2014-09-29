package main

import (
	"testing"
	"fmt"
	"net/http"
	"net/http/httptest"
)

func assertErrors(t *testing.T, r map[string]Page, e error, msg string) {

	if (r != nil) {
		t.Errorf("Wanted nil result " + msg)
	}

	if (e == nil) {
		t.Errorf("Wanted error " + msg)
	}

}

func TestErrorWhenBadUrl(t *testing.T)  {
	r,e := Scan(":?::://www.example.com")

	assertErrors(t, r,e, "scan(badUrl)")

}

func TestErrorWhenNonHttp(t *testing.T) {
	r,e := Scan("htt://www.example.com")

	assertErrors(t, r, e, "scan(nonHttpUrl")
}

func TestErrorWhenUnableToConnect(t *testing.T) {
	r,e := Scan("http://localhost:162000")

	assertErrors(t, r,e, "scan(unableToConnect)")
}

var (
	testhtml = `<!DOCTYPE html>
<html>
<body>

<h1>Test Document</h1>
<p><a href="/link1">My</a> first paragraph.</p>
<p><a _target="some target" class="something" href='/link2'>My</a> first paragraph.</p>
<p><a HREF='http://other.example.com/link3'>My</a> first paragraph.</p>
<p><a href="#sometarget">My</a> first paragraph
<p><a href="javascript:void(0);">bad link</a>

</body>
</html>`

	noLinkTestHtml = `<!DOCTYPE html>
<html>
<head>
	 <script src="/script1.js"></script>
	 <link href="/assets/css/css1.css" media="screen" rel="stylesheet" type="text/css" />
</head>
<body>

<h1>Test Document</h1>
<img src="/assets/images/image1.jpg">

</body>
</html>`

)


func listContains(a string, list []string) int {
	for i,b := range list {
		if (b == a) {
			return i;
		}
	}
	return -1
}

func TestParseHrefs(t *testing.T) {

	links := findHrefs(testhtml)

	correctLinks := []string{"/link1", "/link2", "http://other.example.com/link3", "#sometarget", "javascript:void(0);"}

	if (len(links) != len(correctLinks)) {
		t.Errorf("Found %d links, expecting %d", len(links), len(correctLinks))
	}

	for _, element := range correctLinks  {
		if (listContains(element, links) == -1) {
			t.Errorf("Found missing %s", element)
		}
	}
}

func TestFindLinks(t *testing.T) {
	links := findLinks("http://example.com", testhtml)

	correctLinks := []string{"http://example.com/link1",
			"http://example.com/link2", "http://other.example.com/link3", "http://example.com#sometarget"}

	if (len(links) != len(correctLinks)) {
		t.Errorf("Found %s links, expecting %d", links, len(correctLinks))
	}

	for _, element := range correctLinks  {
		if (listContains(element, links) == -1) {
			t.Errorf("Found missing %s", element)
		}
	}
}

func TestFindResources(t *testing.T) {
	resources := findStaticResources("http://example.com", noLinkTestHtml)

	correctResources := []string{ "http://example.com/assets/images/image1.jpg",
		"http://example.com/script1.js",
		"http://example.com/assets/css/css1.css"}

	if (len(resources) != len(correctResources)) {
		t.Errorf("Found %s resources, expecting %d", resources, len(correctResources))
	}

	for _, element := range correctResources  {
		if (listContains(element, resources) == -1) {
			t.Errorf("Found missing %s", element)
		}
	}
}

func TestGetHtml(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, testhtml)
	}))
	defer ts.Close()

	getHtml(ts.URL, make(semaphore,1))
}

func TestReturnPageWithNoLinks(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, noLinkTestHtml)
	}))
	defer ts.Close()

	pages, _ := Scan(ts.URL)

	if (len(pages) != 1) {
		t.Error("Expected to get back a single page, instead found", pages)
	}

	page, prs := pages[ts.URL]
	if (!prs) {
		t.Error("Expected to get a page back for " + ts.URL)
	}

	if (page.url != ts.URL) {
		t.Errorf("Expected %s, found %s", ts.URL, page.url)
	}

}

func TestHttpError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "failure", http.StatusInternalServerError)
		fmt.Fprintln(w, testhtml)
	}))
	defer ts.Close()

	_, e := getHtml(ts.URL, make(semaphore,1))

	if (e == nil) {
		t.Errorf("Expected Error on 500")
	}
}

func TestDefaultUrlFilter(t *testing.T) {
	defaultFilter := createDefaultUrlFilter("http://example.com")
	if (!defaultFilter("http://example.com/link1")) {
		t.Error("Expected to follow link to same host")
	}
	if (!defaultFilter("https://example.com/link1")) {
		t.Error("Expected to follow link to same host")
	}
	if (defaultFilter("http://other.url.com/link")) {
		t.Error("Expected not to follow link to other host")
	}
}

func TestWholeEnchilada(t *testing.T) {
	//Scan("https://www.digitalocean.com/")
}

