package main

//This package implements a webcrawler. The key function is Scan, which can run on a URL with default options, or
//with a configurable set. The key options are the level of concurrency and also the policy for whether to follow links
//or not.
//Most of the actual work is done in doPageScan, which is recursively called from goroutines for each link on a page
import (
	"fmt"
	"net/url"
	"net/http"
	"io/ioutil"
	"regexp"
	"sync"
	"strings"
	"log"
	"os"
	"time"
)

var logger *log.Logger
var statusLogger *log.Logger
func init() {
	logger = log.New(new(DevNull), "gocrawler: ", log.Lmicroseconds )
	statusLogger = log.New(os.Stdout, "gocrawler: ", log.Lmicroseconds )
}

var (
	//modified version of solution found http://stackoverflow.com/questions/15926142/regular-expression-for-finding-href-value-of-a-a-link
	hrefRegex = "(?i)<a\\s+(?:[^>]*?\\s+)?href=[\",']([^\"']*)[\",']"
	imageRegex = "(?i)<img\\s+(?:[^>]*?\\s+)?src=[\",']([^\"']*)[\",']"
	javascriptRegex = "(?i)<script\\s+(?:[^>]*?\\s+	)?src=[\",']([^\"']*)[\",']"
	cssRegex = "(?i)<link\\s+(?:[^>]*?\\s+	)?href=[\",']([^\"']*)[\",']"
)

type argError struct {
	arg  int
	prob string
}

///Filters URLs.
type UrlFilter func(string)(bool)

//Default filter will restrict URLs to the rootUrl domain. You might also want to add filters to support subdomain
//scanning.
func createDefaultUrlFilter(rootUrl string)(UrlFilter) {
	return func(target string)(bool){
		rootUrl, _ := url.Parse(rootUrl)
		targetUrl, _ := url.Parse(target)
		return rootUrl.Host == targetUrl.Host
	}
}

//Options for domain scanning. urlFilter decides whether to follow links. rootUrl is the opening request
//concurrentRequests decideds how many requests can be run concurrently. Setting this to high values can act like
//a DOS ...
type DomainScanOptions struct {
	urlFilter UrlFilter
	rootUrl string
	concurrentRequests int
}
//Create domain scan options. Defaults to defaultUrl filter and 4 concurrent requests.
func NewDomainScanOptions(rootUrl string)(*DomainScanOptions) {
	options := new(DomainScanOptions)
	options.urlFilter = createDefaultUrlFilter(rootUrl)
	options.rootUrl = rootUrl
	options.concurrentRequests = 4
	return options
}

//Return value from an HtmlGet Request. Right now we only care about the html and the amount of time the request
//took. This would be where we could track more information (http headers for example)
type HtmlResult struct {
	html string
	nanos int64
}

//Url List tracks URLs that have been scanned. This list is used across all concurrent tasks
//and needs to be protected with a mutex to support this.
type UrlList struct {
	urls []string
	mutex sync.RWMutex
}

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

//Unit of result from the scan
type Page struct {
	url string
	outLinks []string
	resources []string
	nanos int64
	parent string
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

//Helper function to traverse regexes
func findAllMatches(re *regexp.Regexp, html string)([]string) {
	matches := re.FindAllStringSubmatch(html, -1)

	links := make([]string,0)
	for i:=0; i<len(matches); i++ {
		match := matches[i]
		links = append(links, match[1])
	}
	return links
}

//resolve relative urls to absolute using root. URLS which are not http(s) are not returned
func resolveTargetUrls(root string, targets []string)([]string) {
	rootUrl, _ := url.Parse(root)

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

//Finds all hrefs using regexs
func findHrefs(html string) ([]string) {

	re := regexp.MustCompile(hrefRegex)
	return findAllMatches(re,html)
}

//Scans using regex for links. Relative links are resolved
func findLinks(root string, html string) ([]string){

	targets := findHrefs(html)
	return resolveTargetUrls(root, targets)
}

//Scans using regex for images/javascripts/css. Relative links are resolved
func findStaticResources(root string, html string)([]string) {
	imageRe := regexp.MustCompile(imageRegex)
	javascriptRe := regexp.MustCompile(javascriptRegex)
	cssRe := regexp.MustCompile(cssRegex)

	imageTargets := findAllMatches(imageRe, html)
	javascriptTargets := findAllMatches(javascriptRe, html)
	cssTargets := findAllMatches(cssRe, html)


	return append(resolveTargetUrls(root, imageTargets),
				  append(resolveTargetUrls(root, javascriptTargets),
					     resolveTargetUrls(root, cssTargets)...)...)
}

//Perform HTTP Get and convert returned body to HTML. If content-type != HTML, returns ""
func getHtml(url string, s semaphore) (HtmlResult, error) {
	//Send a test request

	s.Lock()
	defer s.Unlock()

	start := time.Now()

	resp, err := http.Get(url)

	elapsedNanos := time.Since(start).Nanoseconds()

	if (err != nil) {
		return HtmlResult{"", elapsedNanos}, err
	}

	htmlBytes, _:= ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()

	content_type := http.DetectContentType(htmlBytes)
	var htmlString string
	if (!strings.HasPrefix(content_type, "text/html;")) {
		htmlString = ""
	} else {
		htmlString = string(htmlBytes)
	}


	if (resp.StatusCode != http.StatusOK) {
		return HtmlResult{htmlString, elapsedNanos}, &argError{0, "Server Error"}
	}

	return HtmlResult{htmlString, elapsedNanos}, nil
}

//Area for improvement:
// 1: Tracking parent links. Right now we only store the first parent, but potentially if
//    there are multiple links to a page, we could track them all
// 2: anchor links, right now these are relevant, they shouldn't be.
// 3: Smarter link filter rules. Right now links which are to .jpg are followed
// 4. Error Handling. This version will silently ignore errors from getHtml, we might want to store and report them
func doPageScan(url string, parent string, scannedUrls *UrlList,
	domainScanOptions *DomainScanOptions, output chan Page, s semaphore) {

	//verify that we haven't already scanned this url
	if (scannedUrls.InList(url)) {
		return
	}

	//verify that we should be scanning this url
	if (!domainScanOptions.urlFilter(url)) {
		return
	}

	scannedUrls.AddToList(url)

	logger.Printf("Scanning %s -> %s\n", parent, url)

	htmlResult, err := getHtml(url, s)
	html := htmlResult.html

	if (err != nil) {
		//Todo mark with error
		return
	}

	links := findLinks(url, html)
	resources := findStaticResources(url, html)

	logger.Printf("\tFound %s\n", resources)

	page := Page{url, links, resources, htmlResult.nanos, parent }

	//Kicks off one goroutine for each link found on the page.
	var wg sync.WaitGroup
	wg.Add(len(links))
	for i:=0; i<len(links); i++ {
		l := links[i]
		go func() {
			defer wg.Done()
			doPageScan(l, url, scannedUrls, domainScanOptions, output, s)
		}()
	}
	wg.Wait()

	output <- page
}

//Scans a domain using default scan options
func Scan(url string)(map[string]Page, error)  {
	options := NewDomainScanOptions(url)
	return ScanDomain(options)
}

//Scan a domain specifying scan options
func ScanDomain(options *DomainScanOptions) (map[string]Page, error) {
	url := options.rootUrl
	_, err := validateUrl(url)
	if (err != nil) {
		return nil, err
	}

	//Send a test request
	resp, err := http.Get(url)
	if (err != nil || resp.StatusCode != http.StatusOK) {
		return nil, err
	}

	var scannedUrls UrlList

	output := make(chan Page)
	doneChannel :=make(chan int)

	//collects all the output. Signal on doneChannel when complete
	pages := make(map[string]Page)

	go func() {
		for {
			select {
			case page := <- output:
				statusLogger.Printf("SCANNED: %s in %dms. (linked from %s) \n", page.url, page.nanos/1e6, page.parent)
				statusLogger.Printf("    FOUND: links:%s resources:%s", page.outLinks, page.resources)
				pages[page.url] = page
			case d := <- doneChannel:
				d ++
				return
			}
		}
	}()

	s := make(semaphore, options.concurrentRequests)

	doPageScan(options.rootUrl, "", &scannedUrls, options, output, s)

	doneChannel <- 0

	return pages, nil

}
