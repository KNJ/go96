package go96

import (
	"context"
	"fmt"
	"github.com/sclevine/agouti"
	"log"
	"sync"
)

// Queue ...
type Queue struct {
	pages         []*Navigation
	Workers       int
	ChromeOptions ChromeOptions
}

type ChromeBrowser interface {
	Perform(nav *Navigation)
	Options() ChromeOptions
}

// Navigation ...
type Navigation struct {
	CurrentPage   *Page
	Browser       ChromeBrowser
	chromeOptions *ChromeOptions
	entryURL      string
}

// Page ...
type Page struct {
	*agouti.Page
}

// ChromeOptions
type ChromeOptions struct {
	Args []string
}

// NewQueue ...
func NewQueue(w int) *Queue {
	return &Queue{Workers: w}
}

// SetGlobalChromeOptions ...
func (q *Queue) SetGlobalChromeOptions(co ChromeOptions) *Queue {
	q.ChromeOptions = co
	return q
}

// Add ...
func (q *Queue) Add(url string, browser ChromeBrowser, options *ChromeOptions) {
	nav := &Navigation{
		Browser:       browser,
		chromeOptions: options,
		entryURL:      url,
	}
	q.pages = append(q.pages, nav)
}

var wg sync.WaitGroup

func (q *Queue) Work() {
	ctx, cancel := context.WithCancel(context.Background())
	job := make(chan *Navigation)

	for i := 0; i < q.Workers; i++ {
		wg.Add(1)
		go launch(ctx, job, q)
	}

	for _, nav := range q.pages {
		job <- nav
	}

	cancel()
	wg.Wait()
}

func launch(ctx context.Context, job chan *Navigation, q *Queue) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("worker exit.")
			wg.Done()
			return
		case nav := <-job:
			dequeue(nav, q)
		}
	}
}

func dequeue(nav *Navigation, q *Queue) {
	var driver *agouti.WebDriver
	optionArgs := append(q.ChromeOptions.Args, nav.chromeOptions.Args...)
	if len(optionArgs) == 0 {
		driver = agouti.ChromeDriver()
	} else {
		driver = agouti.ChromeDriver(agouti.ChromeOptions("args", optionArgs))
	}
	if err := driver.Start(); err != nil {
		log.Fatal(err)
	}
	defer driver.Stop()
	page, err := newPage(driver)
	nav.CurrentPage = page
	fmt.Println(nav.entryURL)
	if err != nil {
		log.Fatal(err)
	}
	if err := page.Navigate(nav.entryURL); err != nil {
		log.Fatal(err)
	}
	nav.Browser.Perform(nav)
}

func newPage(driver *agouti.WebDriver) (*Page, error) {
	page, err := driver.NewPage()
	if err != nil {
		return nil, err
	}
	return &Page{page}, nil
}
