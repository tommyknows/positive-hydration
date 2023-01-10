/*
Copied from https://github.com/charmbracelet/bubbles/pull/76/files
 Calendar component

 July                  August                September
 Mo Tu We Th Fr Sa Su  Mo Tu We Th Fr Sa Su  Mo Tu We Th Fr Sa Su
              1  2  3   1  2  3  4  5  6  7            1  2  3  4
  4  5  6  7  8  9 10   8  9 10 11 12 13 14   5  6  7  8  9 10 11
 11 12 13 14 15 16 17  15 16 17 18 19 20 21  12 13 14 15 16 17 18
 18 19 20 21 22 23 24  22 23 24 25 26 27 28  19 20 21 22 23 24 25
 25 26 27 28 29 30 31  29 30 31              26 27 28 29 30
*/

package calendar

import (
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const columnWidth = 3

// Weekday represents a full weekday name and its abbreviation.
type Weekday struct {
	Name         string
	Abbreviation string
}

// EnglishWeekdays.
var EnglishWeekdays = []Weekday{
	{Name: "Monday", Abbreviation: "Mo"},
	{Name: "Tuesday", Abbreviation: "Tu"},
	{Name: "Wednesday", Abbreviation: "We"},
	{Name: "Thursday", Abbreviation: "Th"},
	{Name: "Friday", Abbreviation: "Fr"},
	{Name: "Saturday", Abbreviation: "Sa"},
	{Name: "Sunday", Abbreviation: "Su"},
}

type Event struct {
	Time  time.Time
	Style lipgloss.Style
}

func NewRender(events ...Event) string {
	const monthsDisplayed = 3
	var (
		weekdays = []Weekday{
			{Name: "Monday", Abbreviation: "Mo"},
			{Name: "Tuesday", Abbreviation: "Tu"},
			{Name: "Wednesday", Abbreviation: "We"},
			{Name: "Thursday", Abbreviation: "Th"},
			{Name: "Friday", Abbreviation: "Fr"},
			{Name: "Saturday", Abbreviation: "Sa"},
			{Name: "Sunday", Abbreviation: "Su"},
		}
		//markedDate  = lipgloss.NewStyle().ColorWhitespace((false)).Width(columnWidth).Align(lipgloss.Center).Background(lipgloss.Color("#2782F9"))
		currentDate = lipgloss.NewStyle().ColorWhitespace(false).Width(columnWidth).Align(lipgloss.Center).Background(lipgloss.Color("#BCBCBC")).Foreground(lipgloss.Color("#111111"))

		date = lipgloss.NewStyle().ColorWhitespace(false).Width(columnWidth).Align(lipgloss.Center)
	)
	// Each month will have their days represented in a string array
	calendarMonthRender := make([][]string, monthsDisplayed)
	now := time.Now()

	for monthIndex := range calendarMonthRender {
		firstDayOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

		monthRelativePosition := (monthIndex + 1) - len(calendarMonthRender)
		firstDayOfMonth = firstDayOfMonth.AddDate(0, monthRelativePosition, 0)
		lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, -1)

		// Month name heading and days
		// July                  August                September
		// Mo Tu We Th Fr Sa Su  Mo Tu We Th Fr Sa Su  Mo Tu We Th Fr Sa Su
		s := ""
		s += firstDayOfMonth.Month().String()
		s += "\n"

		// Determine 1st day in the month position in the week to complete with padding
		// Dashes represent the required offset padding (offset * weekday header width):
		// Mo Tu We Th Fr Sa Su
		// ------------ 1  2  3
		//  4  5  6  7  8  9 10
		monthStartingDayOffset := 0
		for i, weekday := range weekdays {
			if weekday.Name == firstDayOfMonth.Weekday().String() {
				monthStartingDayOffset = i
				break
			}
		}
		s += strings.Repeat("   ", monthStartingDayOffset)

		for i := 1; i <= lastDayOfMonth.Day(); i++ {
			var marked bool
			for _, dataPoint := range events {
				yearDiff := (now.Year() - dataPoint.Time.Year()) * 12

				if int(dataPoint.Time.Month())-(int(now.Month())+yearDiff) == monthRelativePosition &&
					i == dataPoint.Time.Day() {
					s += dataPoint.Style.ColorWhitespace(false).Width(columnWidth).Align(lipgloss.Center).Render(strconv.Itoa(i))
					marked = true
					break
				}
			}
			if !marked {
				// Current selected day is highlighted
				if i == now.Day() && monthRelativePosition == 0 {
					s += currentDate.Render(strconv.Itoa(i))
				} else {
					s += date.Render(strconv.Itoa(i))
				}
			}

			// Add a line return on week end to prepare new line
			// Except when the last day in the month ends on the last weekday
			if (i+monthStartingDayOffset)%7 == 0 && i != lastDayOfMonth.Day() {
				s += "\n"
			}
		}

		s += "\n"

		calendarMonthRender[monthIndex] = strings.Split(s, "\n")
	}

	s := ""
	maxNbLine := 0
	for _, month := range calendarMonthRender {
		if len(month) > maxNbLine {
			maxNbLine = len(month)
		}
	}

	// Render calendar by iterating over each month, line by line
	// For example if we have three months, our first line will look like this:
	//        1  2  3  4  5               1  2  3   1  2  3  4  5  6  7
	// The second line will look like this:
	//  6  7  8  9 10 11 12   4  5  6  7  8  9 10   8  9 10 11 12 13 14

	for i := 0; i < maxNbLine; i++ {
		for monthPosition, month := range calendarMonthRender {
			// Padding between months
			if monthPosition > 0 {
				s += " "
			}
			// Render calendar line
			if i < len(month) {
				s += lipgloss.NewStyle().Width(columnWidth * 7).Render(month[i])
			}
		}
		s += "\n"
	}
	return strings.TrimSpace(s)
}

func Render(times []time.Time) string {
	const monthsDisplayed = 3
	var (
		weekdays = []Weekday{
			{Name: "Monday", Abbreviation: "Mo"},
			{Name: "Tuesday", Abbreviation: "Tu"},
			{Name: "Wednesday", Abbreviation: "We"},
			{Name: "Thursday", Abbreviation: "Th"},
			{Name: "Friday", Abbreviation: "Fr"},
			{Name: "Saturday", Abbreviation: "Sa"},
			{Name: "Sunday", Abbreviation: "Su"},
		}
		markedDate  = lipgloss.NewStyle().ColorWhitespace((false)).Width(columnWidth).Align(lipgloss.Center).Background(lipgloss.Color("#2782F9"))
		currentDate = lipgloss.NewStyle().ColorWhitespace(false).Width(columnWidth).Align(lipgloss.Center).Background(lipgloss.Color("#BCBCBC")).Foreground(lipgloss.Color("#111111"))

		date = lipgloss.NewStyle().ColorWhitespace(false).Width(columnWidth).Align(lipgloss.Center)
	)
	// Each month will have their days represented in a string array
	calendarMonthRender := make([][]string, monthsDisplayed)
	now := time.Now()

	for monthIndex := range calendarMonthRender {
		firstDayOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

		monthRelativePosition := (monthIndex + 1) - len(calendarMonthRender)
		firstDayOfMonth = firstDayOfMonth.AddDate(0, monthRelativePosition, 0)
		lastDayOfMonth := firstDayOfMonth.AddDate(0, 1, -1)

		// Month name heading and days
		// July                  August                September
		// Mo Tu We Th Fr Sa Su  Mo Tu We Th Fr Sa Su  Mo Tu We Th Fr Sa Su
		s := ""
		s += firstDayOfMonth.Month().String()
		s += "\n"

		// Determine 1st day in the month position in the week to complete with padding
		// Dashes represent the required offset padding (offset * weekday header width):
		// Mo Tu We Th Fr Sa Su
		// ------------ 1  2  3
		//  4  5  6  7  8  9 10
		monthStartingDayOffset := 0
		for i, weekday := range weekdays {
			if weekday.Name == firstDayOfMonth.Weekday().String() {
				monthStartingDayOffset = i
				break
			}
		}
		s += strings.Repeat("   ", monthStartingDayOffset)

		for i := 1; i <= lastDayOfMonth.Day(); i++ {
			var marked bool
			for _, t := range times {
				yearDiff := (now.Year() - t.Year()) * 12

				if int(t.Month())-(int(now.Month())+yearDiff) == monthRelativePosition &&
					i == t.Day() {
					s += markedDate.Render(strconv.Itoa(i))
					marked = true
					break
				}
			}
			if !marked {
				// Current selected day is highlighted
				if i == now.Day() && monthRelativePosition == 0 {
					s += currentDate.Render(strconv.Itoa(i))
				} else {
					s += date.Render(strconv.Itoa(i))
				}
			}

			// Add a line return on week end to prepare new line
			// Except when the last day in the month ends on the last weekday
			if (i+monthStartingDayOffset)%7 == 0 && i != lastDayOfMonth.Day() {
				s += "\n"
			}
		}

		s += "\n"

		calendarMonthRender[monthIndex] = strings.Split(s, "\n")
	}

	s := ""
	maxNbLine := 0
	for _, month := range calendarMonthRender {
		if len(month) > maxNbLine {
			maxNbLine = len(month)
		}
	}

	// Render calendar by iterating over each month, line by line
	// For example if we have three months, our first line will look like this:
	//        1  2  3  4  5               1  2  3   1  2  3  4  5  6  7
	// The second line will look like this:
	//  6  7  8  9 10 11 12   4  5  6  7  8  9 10   8  9 10 11 12 13 14

	for i := 0; i < maxNbLine; i++ {
		for monthPosition, month := range calendarMonthRender {
			// Padding between months
			if monthPosition > 0 {
				s += " "
			}
			// Render calendar line
			if i < len(month) {
				s += lipgloss.NewStyle().Width(columnWidth * 7).Render(month[i])
			}
		}
		s += "\n"
	}
	return strings.TrimSpace(s)
}
