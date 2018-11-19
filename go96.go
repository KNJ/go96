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
	pages     []Navigation
	Workers   int
	ProxyHost string
	ProxyPort int
}

// Navigation ...
type Navigation interface {
	EntryURL() string
	Perform(page *Page)
}

// Page ...
type Page struct {
	*agouti.Page
}

// NewQueue ...
func NewQueue(w int) *Queue {
	return &Queue{Workers: w}
}

// Configure ...
func (q *Queue) Configure(options ...func(*Queue)) *Queue {
	for _, option := range options {
		option(q)
	}
	return q
}

// Add ...
func (q *Queue) Add(nav Navigation) {
	q.pages = append(q.pages, nav)
}

var wg sync.WaitGroup

// Work ...
func (q *Queue) Work() {
	ctx, cancel := context.WithCancel(context.Background())
	job := make(chan Navigation)

	for i := 0; i < q.Workers; i++ {
		var args []string
		if q.ProxyHost != "" && q.ProxyPort != 0 {
			args = append(args, fmt.Sprintf("--proxy-server=%s:%s", q.ProxyHost, q.ProxyPort))
		}
		driver := agouti.ChromeDriver(agouti.ChromeOptions("args", args))
		if err := driver.Start(); err != nil {
			log.Fatal(err)
		}
		defer driver.Stop()
		wg.Add(1)
		go launch(ctx, job, driver)
	}

	for _, nav := range q.pages {
		job <- nav
	}

	cancel()
	wg.Wait()
}

func launch(ctx context.Context, job chan Navigation, driver *agouti.WebDriver) {
	for {
		select {
		case <-ctx.Done():
			fmt.Println("worker exit.")
			wg.Done()
			return
		case nav := <-job:
			page, err := newPage(driver)
			fmt.Println(nav.EntryURL())
			if err != nil {
				log.Fatal(err)
			}
			if err := page.Navigate(nav.EntryURL()); err != nil {
				log.Fatal(err)
			}
			nav.Perform(page)
		}
	}
}

func newPage(driver *agouti.WebDriver) (*Page, error) {
	page, err := driver.NewPage()
	if err != nil {
		return nil, err
	}
	return &Page{page}, nil
}
