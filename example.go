package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// generate random data for bar chart
func generateBarItems(commits []commitInfo) []opts.BarData {
	items := make([]opts.BarData, 0)
	for _, val := range commits {
		items = append(items, opts.BarData{Value: val.nbrCommits})
	}
	return items
}

func barchart(outputFile string, startDate time.Time, commits []commitInfo) {
	// create a new bar instance
	bar := charts.NewBar()
	// set some global options like Title/Legend/ToolTip or anything else
	bar.SetGlobalOptions(charts.WithTitleOpts(opts.Title{
		Title: "Git Commits powered by go-echarts",
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
	f, _ := os.Create(outputFile)
	bar.Render(f)
}

type commitInfo struct {
	date       time.Time
	nbrCommits int
}

const (
	dateFormat string = "20060102"
)

var (
	repoName   string
	verbose    bool
	quiet      bool
	start, end time.Time
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

func parseDate(input string, startDate bool) (time.Time, error) {
	if startDate {
		input = input + "01"
	} else {
		input = input + "31"
	}
	return time.Parse(dateFormat, input)
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
	commits := make([]commitInfo, totalMonthsBetween)
	totalNbrCommits := 0

	// iterate over the commits
	err = cIter.ForEach(func(c *object.Commit) error {

		totalNbrCommits++

		if verbose {
			fmt.Println(c)
			fmt.Println(c.Author.When.Month())
		}

		monthSlot := monthsBetween(start, c.Author.When)
		if monthSlot > 0 {
			commits[monthSlot-1].nbrCommits++
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
		}

		return nil
	})
	CheckIfError(err)

	if !quiet {
		fmt.Printf("processed %d commits over %d months\n", totalNbrCommits, totalMonthsBetween)
	}
	if verbose {
		for index, commit := range commits {
			fmt.Printf("%d: %d commits\n", index, commit.nbrCommits)
		}
	}

	barchart(outputFile, start, commits)
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
