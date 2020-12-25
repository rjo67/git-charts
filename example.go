package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/components"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func generatePieItems(commits []commitInfo) []opts.PieData {
	data := make(map[string]int)

	// iterate over commits and retrieve author info
	totalCommits := 0
	for _, commit := range commits {
		for author, nbrCommits := range commit.authors {
			//fmt.Printf("author, commits: %s:\t%d\n", author, nbrCommits)
			data[author] += nbrCommits
			totalCommits += nbrCommits
		}
	}
	fmt.Printf("totalCommits: %d\n", totalCommits)

	items := make([]opts.PieData, 0)
	for k, val := range data {
		items = append(items, opts.PieData{Name: k, Value: val})
	}
	return items
}

func pieBase(commits []commitInfo, timeFrame string, totalCommits int) *charts.Pie {
	pie := charts.NewPie()
	pie.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{Title: fmt.Sprintf("Commits per author (%s in total) (%s)", totalCommits, timeFrame)}),
	)

	pie.AddSeries("pie", generatePieItems(commits))
	return pie
}

// generate random data for bar chart
func generateBarItems(commits []commitInfo) []opts.BarData {
	items := make([]opts.BarData, 0)
	for _, val := range commits {
		items = append(items, opts.BarData{Value: val.nbrCommits})
	}
	return items
}

func barchart(startDate time.Time, commits []commitInfo, timeFrame string, totalCommits int) *charts.Bar {
	// create a new bar instance
	bar := charts.NewBar()
	// set some global options like Title/Legend/ToolTip or anything else
	bar.SetGlobalOptions(charts.WithTitleOpts(opts.Title{
		Title: fmt.Sprintf("Git commits per month (%d in total) (%s)", totalCommits, timeFrame),
	}))

	// create X-Axis (months)
	months := make([]string, 0)
	for cnt := 0; cnt < len(commits); startDate = startDate.AddDate(0, 1, 0) {
		cnt++
		months = append(months, startDate.Format("Jan-06"))
	}

	// Put data into instance
	bar.SetXAxis(months).
		AddSeries("Category A", generateBarItems(commits))
		//AddSeries("Category B", generateBarItems())

	return bar
}

// stores information about a set of commits
type commitInfo struct {
	date       time.Time
	nbrCommits int
	authors    map[string]int
}

func makeCommitData(size int) []commitInfo {
	commits := make([]commitInfo, size)
	commits[0].authors = make(map[string]int)
	for slot := range commits {
		commits[slot].authors = make(map[string]int)
	}
	return commits
}

const (
	dateFormat string = "20060102 15:04:05"
)

var (
	repoName   string
	verbose    bool
	quiet      bool
	start, end time.Time
	timeFrame  string // string representation of start..end
	outputFile string // name of the chart output file
)

func openRepo(path string) (*git.Repository, error) {
	return git.PlainOpen(path)
}

// monthsBetween returns the number of months between the given start and end dates
func monthsBetween(start, end time.Time) int {
	nbrMonths := 0
	for ; end.After(start); start = start.AddDate(0, 1, 0) {
		nbrMonths++
	}
	return nbrMonths
}

// parseDate expects a 4-digit input string (year and month) and adds on the first or last day of the month.
// An object representing this date is returned
func parseDate(input string, startDate bool) (time.Time, error) {
	startOfMonth, err := time.Parse(dateFormat, input+"01 00:00:00")
	if err != nil {
		return startOfMonth, err
	}
	if startDate {
		return startOfMonth, err
	}
	endOfMonth, err := time.Parse(dateFormat, input+"01 23:59:59")
	if err != nil {
		return endOfMonth, err
	}
	endOfMonth = endOfMonth.AddDate(0, 1, -1)
	return endOfMonth, err
}

func parseParameters() error {
	var startStr, endStr string

	flag.StringVar(&repoName, "r", "", "repository name (required)")
	flag.StringVar(&startStr, "s", "", "start date (format yyyymm)")
	flag.StringVar(&endStr, "e", "", "end date (format yyyymm) (optional, defaults to current month)")
	flag.StringVar(&outputFile, "o", "output.html", "output filename")
	flag.BoolVar(&verbose, "v", false, "verbose")
	flag.BoolVar(&quiet, "q", false, "extra quiet")

	flag.Parse()

	if repoName == "" {
		return errors.New("repository name not specified")
	}

	if len(startStr) != 6 {
		return errors.New("wrong format for start date (must be 4 chars)")
	}
	if endStr == "" {
		// use current date
		endStr = time.Now().Format("200601")
	} else if len(endStr) != 6 {
		return errors.New("wrong format for end date (must be 4 chars)")
	}
	var err error
	start, err = parseDate(startStr, true)
	if err != nil {
		return err
	}
	end, err = parseDate(endStr, false)
	if err != nil {
		return err
	}
	//	fmt.Printf("got start: %s, end: %s\n", start, end)

	if end.Before(start) {
		return errors.New("end date is before start date")
	}

	return nil
}

func main() {

	startTime := time.Now()

	err := parseParameters()
	CheckIfError(err)

	repo, err := openRepo(repoName)
	CheckIfError(err)
	if !quiet {
		fmt.Printf("opened repo %s\n", repoName)
	}

	ref, err := repo.Head()
	CheckIfError(err)

	// ... retrieves the commit history
	cIter, err := repo.Log(&git.LogOptions{From: ref.Hash(), Since: &start, Until: &end})
	CheckIfError(err)

	totalMonthsBetween := monthsBetween(start, end)
	timeFrame := fmt.Sprintf("from %s to %s", start.Format(dateFormat), end.Format(dateFormat))
	commits := makeCommitData(totalMonthsBetween)
	totalNbrCommits := 0

	// iterate over the commits
	err = cIter.ForEach(func(c *object.Commit) error {

		monthSlot := monthsBetween(start, c.Author.When)
		if monthSlot > 0 {

			totalNbrCommits++

			/*
				if verbose {
					fmt.Println(c)
				}
			*/

			commits[monthSlot-1].nbrCommits++
			commits[monthSlot-1].authors[c.Author.Name]++
			/*
				 getting the commit stats is very expensive...

				stats, err := c.Stats()
				CheckIfError(err)

				if verbose {
					fmt.Println("Stats:")
					for index, stat := range stats {
						fmt.Printf("%d: +%d, -%d\n", index, stat.Addition, stat.Deletion)
					}
					fmt.Println("---------")
				}
			*/

		} else {
			if verbose {
				fmt.Printf("ignoring commit outside of required date: %s, %s\n", c.Hash, c.Author.When)
			}
		}

		return nil
	})
	CheckIfError(err)

	if !quiet {
		fmt.Printf("processed %d commits over %d months\n", totalNbrCommits, totalMonthsBetween)
	}
	if verbose {
		fmt.Println("Processed following commits:")
		for index, commit := range commits {
			fmt.Printf("%d: %d commits\n", index, commit.nbrCommits)
		}
	}

	page := components.NewPage()
	page.AddCharts(
		barchart(start, commits, timeFrame, totalNbrCommits),
		pieBase(commits, timeFrame, totalNbrCommits),
	)
	f, err := os.Create(outputFile)
	CheckIfError(err)
	page.Render(io.MultiWriter(f))

	if !quiet {
		fmt.Printf("output in %s\n", outputFile)
	}

	if !quiet {
		fmt.Printf("finished in %s\n", time.Since(startTime))
	}
}

// CheckIfError should be used to naively panic if an error is not nil.
func CheckIfError(err error) {
	if err == nil {
		return
	}

	fmt.Printf("\x1b[31;1m%s\x1b[0m\n", fmt.Sprintf("error: %s", err))
	os.Exit(1)
}
