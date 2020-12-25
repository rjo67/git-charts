package main

import (
	"testing"
)

func TestCommitData(t *testing.T) {
	c := makeCommitData(5)
	c[0].nbrCommits++
	c[0].authors["rjo"]++
}

func TestMonthsBetween(t *testing.T) {

	data := []struct {
		startDate, endDate string
		expectedMonths     int
	}{
		{"201701", "201705", 5},
		{"202001", "202012", 12},

		{"202012", "202012", 1},
		{"202012", "202101", 2},

		{"202001", "202101", 13},
		{"202001", "202112", 24},
		// edge cases
		{"201701", "201612", 0},
		{"201701", "201702", 2}, // Feb only has 28/29 days
		{"201702", "201704", 3}, // Apr only has 30 days
	}

	for _, test := range data {
		start, err := parseDate(test.startDate, true)
		if err != nil {
			t.Errorf("error: got %s parsing date: %s)", err, test.startDate)
		}
		end, err := parseDate(test.endDate, false)
		if err != nil {
			t.Errorf("error: got %s parsing date: %s)", err, test.endDate)
		}
		months := monthsBetween(start, end)
		if months != test.expectedMonths {
			t.Errorf("error: got %d, expected %d (input dates: %s(%s), %s(%s))", months, test.expectedMonths, start, test.startDate, end, test.endDate)
		}
	}
}
