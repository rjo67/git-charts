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

func generatePieItems(data programData) ([]opts.PieData, int, int) {
	mydata := make(map[string]int)
	authorsBelowThreshold := 0

	totalCommits := 0 // just for sanity check
	// merge per-month commit/author info into mydata
	for _, commit := range data.commits {
		for author, nbrCommits := range commit.authors {
			//fmt.Printf("author, commits: %s:\t%d\n", author, nbrCommits)
			mydata[author] += nbrCommits
			totalCommits += nbrCommits
		}
	}
	if totalCommits != data.totalCommits {
		fmt.Printf("Warning: commits for pie chart (%d) != totalCommits (%d)!\n", totalCommits, data.totalCommits)
	}

	totalNbrAuthors := len(mydata)

	// iterate over mydata and combine entries < threshold into one entry
	for author, val := range mydata {
		if val < data.threshold {
			mydata["___others"] += val
			delete(mydata, author)
			authorsBelowThreshold++
			//fmt.Printf("deleted author %s with %d commits\n", author, val)
		}
	}

	items := make([]opts.PieData, 0)
	for k, val := range mydata {
		items = append(items, opts.PieData{Name: k, Value: val})
	}
	return items, totalNbrAuthors, authorsBelowThreshold
}

func pieBase(data programData) *charts.Pie {
	pie := charts.NewPie()
	pieData, totalNbrAuthors, authorsBelowThreshold := generatePieItems(data)

	pie.SetGlobalOptions(charts.WithTitleOpts(opts.Title{
		Title:    "Commits per author",
		Subtitle: fmt.Sprintf("%d commits, %d authors (%d below threshold '%d')", data.totalCommits, totalNbrAuthors, authorsBelowThreshold, data.threshold),
	}))

	pie.AddSeries("pie", pieData).SetSeriesOptions(charts.WithLabelOpts(
		opts.Label{
			Show:      true,
			Formatter: "{b}: {c}",
		}),
	)
	return pie
}

// generate commit-data for bar chart
func generateCommitBarItems(data programData) []opts.BarData {
	items := make([]opts.BarData, 0)
	for _, val := range data.commits {
		items = append(items, opts.BarData{Value: val.nbrCommits})
	}
	return items
}

// generate author-data for bar chart
func generateAuthorBarItems(data programData) []opts.BarData {
	items := make([]opts.BarData, 0)
	for _, val := range data.commits {
		items = append(items, opts.BarData{Value: len(val.authors)})
	}
	return items
}
func barchart(data programData) *charts.Bar {
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title:    "Git commits per month",
			Subtitle: fmt.Sprintf("%d commits, %s", data.totalCommits, data.timeFrame),
		}),
		charts.WithDataZoomOpts(opts.DataZoom{
			Type:  "slider",
			Start: 0,
			End:   100,
		}),
	)

	// create X-Axis (months)
	months := make([]string, 0)
	startDate := data.start
	for cnt := 0; cnt < len(data.commits); startDate = startDate.AddDate(0, 1, 0) {
		cnt++
		months = append(months, startDate.Format("Jan-06"))
	}

	// Put data into instance
	bar.SetXAxis(months).
		AddSeries("Commits", generateCommitBarItems(data)).
		//		, charts.WithBarChartOpts(
		//			opts.BarChart{YAxisIndex: 0},
		//		)).
		AddSeries("Authors", generateAuthorBarItems(data)).
		SetSeriesOptions(
			charts.WithLabelOpts(opts.Label{
				Show:     true,
				Position: "top",
			}),
		)
	bar.SetSeriesOptions(charts.WithMarkLineNameTypeItemOpts(
		opts.MarkLineNameTypeItem{Name: "Maximum", Type: "max"},
		opts.MarkLineNameTypeItem{Name: "Avg", Type: "average"},
	))

	return bar
}

// programData stores all info required to render the charts
type programData struct {
	start, end   time.Time
	timeFrame    string // string representation of start..end
	threshold    int    // below this, will group commits or authors instead of listing them singly
	totalCommits int    // total commits found in the time frame
	commits      []commitInfo
}

// commitInfo stores information about a set of commits (e.g. per month)
type commitInfo struct {
	nbrCommits int
	authors    map[string]int
}

func makeCommitData(size int) []commitInfo {
	commits := make([]commitInfo, size)
	for slot := range commits {
		commits[slot].authors = make(map[string]int)
	}
	return commits
}

const (
	dateFormat string = "20060102 15:04:05"
)

var (
	outputFile string // name of the chart output file
	data       programData
	quiet      bool
	repoName   string
	verbose    bool
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

	data = programData{}

	flag.StringVar(&repoName, "r", "", "repository name (required)")
	flag.StringVar(&startStr, "s", "", "start date (format yyyymm)")
	flag.StringVar(&endStr, "e", "", "end date (format yyyymm) (optional, defaults to current month)")
	flag.StringVar(&outputFile, "o", "output.html", "output filename")
	flag.BoolVar(&verbose, "v", false, "verbose")
	flag.BoolVar(&quiet, "q", false, "extra quiet")
	flag.IntVar(&data.threshold, "t", 3, "threshold for commits")

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
	data.start, err = parseDate(startStr, true)
	if err != nil {
		return err
	}
	data.end, err = parseDate(endStr, false)
	if err != nil {
		return err
	}
	//	fmt.Printf("got start: %s, end: %s\n", start, end)

	if data.end.Before(data.start) {
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
	cIter, err := repo.Log(&git.LogOptions{From: ref.Hash(), Since: &data.start, Until: &data.end})
	CheckIfError(err)

	totalMonthsBetween := monthsBetween(data.start, data.end)
	data.timeFrame = fmt.Sprintf("from %s to %s", data.start.Format("2006-01-02"), data.end.Format("2006-01-02"))
	data.commits = makeCommitData(totalMonthsBetween)

	// iterate over the commits
	err = cIter.ForEach(func(c *object.Commit) error {

		monthSlot := monthsBetween(data.start, c.Author.When)
		if monthSlot > 0 {

			data.totalCommits++

			/*
				if verbose {
					fmt.Println(c)
				}
			*/

			data.commits[monthSlot-1].nbrCommits++
			data.commits[monthSlot-1].authors[c.Author.Name]++
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
		fmt.Printf("processed %d commits over %d months\n", data.totalCommits, totalMonthsBetween)
	}
	if verbose {
		fmt.Println("Processed following commits:")
		for index, commit := range data.commits {
			fmt.Printf("%d: %d commits\n", index, commit.nbrCommits)
		}
	}

	page := components.NewPage()
	page.AddCharts(
		barchart(data),
		pieBase(data),
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
