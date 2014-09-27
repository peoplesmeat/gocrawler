package gocrawler

import (
	"fmt"
	"net/url"
	"net/http"
	"regexp"
)

type argError struct {
	arg  int
	prob string
}

type domainScan struct {
}


func (e *argError) Error() string {
	return fmt.Sprintf("%d - %s", e.arg, e.prob)
}

func validateUrl(domainName string) (*url.URL, error) {
	u, err := url.Parse(domainName)

	if (err != nil) {
		return nil, &argError{0, "Not a URL"}
	}

	if (u.Scheme != "http" && u.Scheme != "https") {
		return nil, &argError{0, "Bad Scheme"}
	}

	return u, nil

}

func findLinks(html string) ([]string) {
	//modified version of solution found http://stackoverflow.com/questions/15926142/regular-expression-for-finding-href-value-of-a-a-link
	re := regexp.MustCompile("(?i)<a\\s+(?:[^>]*?\\s+)?href=[\",']([^\"']*)[\",']")
	matches := re.FindAllStringSubmatch(html, -1)

	links := make([]string,0)
	for i:=0; i<len(matches); i++ {
		match := matches[i]
		links = append(links, match[1])
	}
	return links
}

func scan(domainName string) (*domainScan, error) {

	u, err := validateUrl(domainName)
	if (err != nil) {
		return nil, err
	}

	//Send a test request
	resp, err := http.Get(domainName)
	if (err != nil) {
		return nil, err
	}

	fmt.Println("returning nil error:" + u.Scheme + ":" + u.Host + ":" + resp.Status)
	var _ = u
	var _ = resp

	return nil, nil

}
