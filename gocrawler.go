package main

import (
	"fmt"
	"net/url"
	"net/http"
	"io/ioutil"
	"regexp"
	"sync"
)

type argError struct {
	arg  int
	prob string
}

type UrlFilter func(string)(bool)

type DomainScanOptions struct {
	urlFilter UrlFilter
}

func NewDomainScanOptions(rootUrl string)(*DomainScanOptions) {
	options := new(DomainScanOptions)
	options.urlFilter = func(target string)(bool){
		rootUrl, _ := url.Parse(rootUrl)
		targetUrl, _ := url.Parse(target)
		return rootUrl.Host == targetUrl.Host
	}

	return options
}

type DomainScan struct {
	options DomainScanOptions
}

type UrlList struct {
	urls []string
	mutex sync.RWMutex
}

type Page struct {
	url string
	staticResources []string
	linksTo []string
}

/*func NewUrlList() *UrlList {
	urlList := UrlList{make([]string, 0)}
	return &urlList
}*/

func (list *UrlList) InList(url string) bool {
	list.mutex.RLock()
	defer list.mutex.RUnlock()

	for i:=0; i<len(list.urls); i++ {
		if (list.urls[i] == url) {
			return true
		}
	}

	return false
}
func (list *UrlList) AddToList(url string) {
	list.mutex.Lock()
	defer list.mutex.Unlock()

	list.urls = append(list.urls, url)
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

func findHrefs(html string) ([]string) {
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

func findLinks(root string, html string) ([]string){
	rootUrl, _ := url.Parse(root)

	targets := findHrefs(html)

	resolvedTargets := make([]string,0)
	for i:=0;i<len(targets);i++ {
		targetUrl,_ := url.Parse(targets[i])

		if (targetUrl.Scheme == "http" || targetUrl.Scheme == "https" || targetUrl.Scheme == "") {
			// If ref is an absolute URL, then ResolveReference ignores base and returns a copy of ref.
			resolvedTargets = append(resolvedTargets, rootUrl.ResolveReference(targetUrl).String())
		}

	}

	return resolvedTargets
}

func findStaticResources(html string)([]string) {
	return make([]string, 0)
}

func getHtml(url string) (string, error) {
	//Send a test request
	resp, err := http.Get(url)

	if (err != nil) {
		return "", err
	}

	html_bytes, _:= ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	html := string(html_bytes)

	if (resp.StatusCode != http.StatusOK) {
		return html, &argError{0, "Server Error"}
	}

	return html, nil
}

func doPageScan(url string, parent string, scannedUrls *UrlList, domainScanOptions *DomainScanOptions)([]Page) {

	//verify that we haven't already scanned this url
	if (scannedUrls.InList(url)) {
		return make([]Page,0)
	}

	//verify that we should be scanning this url
	if (!domainScanOptions.urlFilter(url)) {
		return make([]Page,0)
	}

	scannedUrls.AddToList(url)

	fmt.Printf("Scanning %s -> %s\n", parent, url)

	html, err := getHtml(url)
	if (err != nil) {
		//Todo mark with error
		return make([]Page,0)
	}

	links := findLinks(url, html)

	var page Page
	page.url = url
	page.linksTo = links

	pages := []Page{page}

	for i:=0; i<len(links); i++ {
		//check if already scanned
		childPages := doPageScan(links[i], url, scannedUrls, domainScanOptions)
		pages = append(pages, childPages...)
	}

	return pages
}

func Scan(url string) (*DomainScan, error) {

	u, err := validateUrl(url)
	if (err != nil) {
		return nil, err
	}

	//Send a test request
	resp, err := http.Get(url)
	if (err != nil) {
		return nil, err
	}

	fmt.Println("returning nil error:" + u.Scheme + ":" + u.Host + ":" + resp.Status)

	var scannedUrls UrlList

	options := NewDomainScanOptions(url)

	doPageScan(url, "", &scannedUrls, options)

	return nil, nil

}
