package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	calendar "github.com/tommyknows/positive-hydration/calendar"
)

const dbFile = "/Users/ramon/.config/positive_hydration.json"

var (
	titleStyle        = lipgloss.NewStyle().MarginLeft(2).Bold(true)
	itemStyle         = lipgloss.NewStyle().PaddingLeft(4)
	selectedItemStyle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	paginationStyle   = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	baseStyle         = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("240"))
	quitTextStyle     = lipgloss.NewStyle().Margin(1, 0, 2, 4)

	focusedStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	blurredStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	cursorStyle         = focusedStyle.Copy()
	noStyle             = lipgloss.NewStyle()
	cursorModeHelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))

	focusedButton = focusedStyle.Copy().Render("[ Submit ]")
	blurredButton = fmt.Sprintf("[ %s ]", blurredStyle.Render("Submit"))

	boxed = lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1, 0)
)

type ShowPlants struct {
	*PlantDB
	showPlant *Plant
	list      list.Model

	prompt tea.Model
}

func newShowPlants(pDB *PlantDB) *ShowPlants {
	items := make([]list.Item, 0, len(pDB.Plants))
	for _, plant := range pDB.Plants {
		items = append(items, plant)
	}

	// sort plants by next watering day.
	sort.Slice(items, func(i, j int) bool {
		di, iok := scheduledIn(last(items[i].(*Plant).WateredAt), items[i].(*Plant).WateringIntervals)
		dj, jok := scheduledIn(last(items[j].(*Plant).WateredAt), items[j].(*Plant).WateringIntervals)
		if iok && jok {
			return di < dj
		} else if jok {
			return true
		}
		return false
	})

	delegate := list.NewDefaultDelegate()
	delegate.SetHeight(3)
	l := list.New(items, delegate, 80, 31)
	// overwrite nextPage keys as "f" is used to mark as fertilized.
	var keys []string
	for _, k := range l.KeyMap.NextPage.Keys() {
		if k != "f" {
			keys = append(keys, k)
		}
	}
	l.KeyMap.NextPage.SetKeys(keys...)

	l.Title = "Your Glorious Plants"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)
	l.Styles.Title = titleStyle
	l.Styles.HelpStyle = list.DefaultStyles().HelpStyle.PaddingLeft(4)
	l.Styles.PaginationStyle = paginationStyle
	l.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
			key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "water")),
			key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "fertilized")),
			key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "repotted")),
			key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
		}
	}
	l.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			key.NewBinding(
				key.WithKeys("a"),
				key.WithHelp("a", "add plant"),
			),
			key.NewBinding(
				key.WithKeys("w"),
				key.WithHelp("w", "mark as watered"),
			),
			key.NewBinding(
				key.WithKeys("W"),
				key.WithHelp("W", "mark as watered with specific date"),
			),
			key.NewBinding(
				key.WithKeys("f"),
				key.WithHelp("f", "mark as fertilized"),
			),
			key.NewBinding(
				key.WithKeys("p"),
				key.WithHelp("p", "mark as repotted"),
			),
			key.NewBinding(
				key.WithKeys("e"),
				key.WithHelp("e", "edit plant"),
			),
		}
	}

	return &ShowPlants{
		PlantDB: pDB,
		list:    l,
	}
}

func (sp *ShowPlants) View() string {
	if len(sp.Plants) == 0 {
		return quitTextStyle.Render("You have no planties yet! Press a to add")
	}

	if len(sp.list.VisibleItems()) > 0 {
		sp.showPlant = sp.list.VisibleItems()[sp.list.Index()].(*Plant)
	}

	right := sp.showPlant.Render(!sp.list.Help.ShowAll)
	if sp.prompt != nil {
		right = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1, 0).
			Height(lipgloss.Height(right)-2). // for borders, I think
			Width(lipgloss.Width(right)-2).   // for padding
			Align(lipgloss.Left, lipgloss.Top).
			Render(sp.prompt.View())
	}

	sp.list.Help.Width = lipgloss.Width(right) - 2
	right = lipgloss.JoinVertical(lipgloss.Center, right,
		lipgloss.NewStyle().Height(31-lipgloss.Height(right)).Align(lipgloss.Center, lipgloss.Bottom).Render(sp.list.Help.View(sp.list)),
	)

	return lipgloss.JoinHorizontal(lipgloss.Top,
		boxed.Copy().Width(40).Render(sp.list.View()),
		boxed.Render(right),
	)
}

func (sp *ShowPlants) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// TODO, not sure what to do.
		return sp, nil

	case tea.KeyMsg:
		switch keypress := msg.String(); keypress {
		case "ctrl+c":
			return sp, tea.Quit

		case "w", "f", "p", "W", "e":
			if sp.list.SettingFilter() || sp.prompt != nil {
				break
			}
			if len(sp.list.VisibleItems()) > 0 {
				p := sp.list.VisibleItems()[sp.list.Index()].(*Plant)
				switch keypress {
				case "w":
					p.WateredAt = toggleEvent(p.WateredAt, time.Now())
				case "W":
					sp.prompt = newWateringPrompt(p)
					return sp, nil
				case "f":
					sp.prompt = newFertilizerPrompt(p)
					return sp, nil
				case "p":
					sp.prompt = newRepottingPrompt(p)
					return sp, nil
				case "e": // edit
					sp.prompt = p.Prompt(nil)
					return sp, nil
				}
			}

		case "a":
			if sp.list.SettingFilter() || sp.prompt != nil {
				break
			}
			// TODO: list doesn't actually update after adding the plant.
			// App has to be restarted for that.
			var p *Plant
			sp.prompt = p.Prompt(func(p *Plant) {
				sp.PlantDB.Plants = append(sp.PlantDB.Plants, p)
			})
			return sp, nil

		case "esc":
			if !sp.list.IsFiltered() && !sp.list.SettingFilter() && sp.prompt == nil {
				return sp, tea.Quit
			}
			if sp.prompt != nil {
				sp.prompt = nil
				return sp, nil
			}

		default:
			if !sp.list.SettingFilter() && sp.prompt == nil && keypress == "q" {
				return sp, tea.Quit
			}
		}
	}
	var cmd tea.Cmd
	if sp.prompt != nil {
		sp.prompt, cmd = sp.prompt.Update(msg)
		return sp, cmd
	}

	sp.list, cmd = sp.list.Update(msg)
	return sp, cmd
}

func (sp *ShowPlants) Init() tea.Cmd {
	return nil
}

type inputPrompt struct {
	focusIndex    int
	inputs        []textinput.Model
	confirmAction func(*inputPrompt) tea.Model
	title         string
}

func (ip *inputPrompt) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render(ip.title) + ":\n\n")

	for i := range ip.inputs {
		b.WriteString(ip.inputs[i].View())
		if i < len(ip.inputs)-1 {
			b.WriteRune('\n')
		}
	}

	button := &blurredButton
	if ip.focusIndex == len(ip.inputs) {
		button = &focusedButton
	}
	fmt.Fprintf(&b, "\n\n%s\n\n", *button)

	return b.String()
}
func (ip *inputPrompt) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(ip.inputs))

	// Only text inputs with Focus() set will respond, so it's safe to simply
	// update all of them here without any further logic.
	for i := range ip.inputs {
		ip.inputs[i], cmds[i] = ip.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (ip *inputPrompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return ip, tea.Quit
		case "esc":
			return nil, nil

		// Set focus to next input
		case "tab", "shift+tab", "enter", "up", "down":
			s := msg.String()

			// Did the user press enter while the submit button was focused?
			// If so, exit.
			if s == "enter" && ip.focusIndex == len(ip.inputs) {
				return ip.confirmAction(ip), nil
			}

			// Cycle indexes
			if s == "up" || s == "shift+tab" {
				ip.focusIndex--
			} else {
				ip.focusIndex++
			}

			if ip.focusIndex > len(ip.inputs) {
				ip.focusIndex = 0
			} else if ip.focusIndex < 0 {
				ip.focusIndex = len(ip.inputs)
			}

			cmds := make([]tea.Cmd, len(ip.inputs))
			for i := 0; i <= len(ip.inputs)-1; i++ {
				if i == ip.focusIndex {
					// Set focused state
					cmds[i] = ip.inputs[i].Focus()
					ip.inputs[i].PromptStyle = focusedStyle
					ip.inputs[i].TextStyle = focusedStyle
					continue
				}
				// Remove focused state
				ip.inputs[i].Blur()
				ip.inputs[i].PromptStyle = noStyle
				ip.inputs[i].TextStyle = noStyle
			}

			return ip, tea.Batch(cmds...)
		}
	}

	// Handle character input and blinking
	return ip, ip.updateInputs(msg)
}
func (ip *inputPrompt) Init() tea.Cmd { return textinput.Blink }

func (p *Plant) Prompt(confirm func(p *Plant)) *inputPrompt {
	if p == nil && confirm == nil {
		return nil
	}

	withValue := func(ti textinput.Model, value string) textinput.Model {
		if p != nil {
			ti.SetValue(value)
			// set value also sets focus, so remove again.
			ti.Blur()
		}
		return ti
	}

	var (
		plantName   = newTextInput("Plant Name", "Friedrich")
		variety     = newTextInput("Variety", "monstera deliciosa")
		location    = newTextInput("Location", "Kitchen")
		wetSoil     = newIntInput("Wet Soil Depth", "in cm")
		watering    = newIntervalInput("Watering Intervals")
		fertilizing = newIntervalInput("Fertilizing Intervals")
		potSize     = newIntInput("Pot Size", "in cm")
		lightLevel  = newLightLevelInput()
		sourcedFrom = newTextInput("Sourced From", "Propagation")
		comments    = newTextInput("Comments", "...")
	)

	if p != nil {
		plantName = withValue(plantName, p.Name)
		variety = withValue(variety, p.Variety)
		location = withValue(location, p.Location)
		wetSoil = withValue(wetSoil, strconv.Itoa(p.WetSoilDepth))
		watering = withValue(watering, p.WateringIntervals.String())
		fertilizing = withValue(fertilizing, p.FertilizingIntervals.String())
		potSize = withValue(potSize, strconv.Itoa(p.PotSize))
		lightLevel = withValue(lightLevel, p.LightLevel.String())
		sourcedFrom = withValue(sourcedFrom, p.SourcedFrom)
		comments = withValue(comments, p.Comments)
	}
	plantName.Focus()
	plantName.PromptStyle = focusedStyle
	plantName.TextStyle = focusedStyle

	title := "Edit Plant"
	if p == nil {
		p = new(Plant)
		title = "Add Plant"
	}
	return &inputPrompt{
		title: title,
		inputs: []textinput.Model{
			plantName, variety, location,
			wetSoil, watering, fertilizing, potSize,
			lightLevel, sourcedFrom, comments,
		},
		confirmAction: func(ap *inputPrompt) tea.Model {
			p.Name = ap.inputs[0].Value()
			p.Variety = ap.inputs[1].Value()
			p.Location = ap.inputs[2].Value()
			p.WetSoilDepth = func() int {
				s, _ := strconv.Atoi(ap.inputs[3].Value())
				return s
			}()
			p.WateringIntervals = func() SeasonalIntervals {
				si, _ := parseSeasonalIntervals(ap.inputs[4].Value())
				return si
			}()
			p.FertilizingIntervals = func() SeasonalIntervals {
				si, _ := parseSeasonalIntervals(ap.inputs[5].Value())
				return si
			}()
			p.PotSize = func() int {
				s, _ := strconv.Atoi(ap.inputs[6].Value())
				return s
			}()
			p.LightLevel = func() *LightLevel {
				l, _ := parseLightLevel(ap.inputs[7].Value())
				return l
			}()
			p.SourcedFrom = ap.inputs[8].Value()
			p.Comments = ap.inputs[9].Value()
			if confirm != nil {
				confirm(p)
			}
			return nil
		},
	}
}

func newTextInput(prompt, placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Prompt = prompt + " > "
	ti.CursorStyle = cursorStyle
	ti.CharLimit = 1000
	ti.Placeholder = placeholder
	return ti
}
func newIntInput(prompt, placeholder string) textinput.Model {
	ti := newTextInput(prompt, placeholder)
	ti.Validate = func(s string) error {
		_, err := strconv.Atoi(s)
		return err
	}
	return ti
}
func newIntervalInput(prompt string) textinput.Model {
	ti := newTextInput(prompt, "summer/winter, summer, summer/-")
	ti.Validate = func(s string) error {
		_, err := parseSeasonalIntervals(s)
		return err
	}
	return ti
}
func newLightLevelInput() textinput.Model {
	placeholder := "0 - direct, 1 - bright, 2 - semi-shaded, 3 - shaded"
	ti := newTextInput("Light Level", placeholder)

	ti.Validate = func(s string) error {
		_, err := parseLightLevel(s)
		return err
	}

	return ti
}
func newDateInput(prompt, placeholder string) textinput.Model {
	ti := newTextInput(prompt, placeholder)
	today := time.Now()
	ti.SetValue(today.Format("2006-01-02"))
	ti.Blur() // SetValue seems to also set focus?
	ti.Validate = func(s string) error {
		_, err := parseInputDate(s)
		return err
	}
	return ti
}

// we extract the current sections & "autocomplete" them.
// With that, we parse the time and check if it's valid.
func parseInputDate(s string) (time.Time, error) {
	// needs to be "any" so that we can spread it to fmt.Sprintf below.
	expectedSections := []any{01, 01, 01}
	for i, sec := range strings.Split(s, "-") {
		if sec == "" || sec == "0" {
			continue
		}
		val, err := strconv.Atoi(sec)
		if err != nil {
			return time.Time{}, err
		}
		expectedSections[i] = val
	}
	t, err := time.Parse("2006-01-02", fmt.Sprintf("%04v-%02v-%02v", expectedSections...))
	if err != nil {
		return time.Time{}, err
	}
	if t.After(time.Now()) {
		return time.Time{}, fmt.Errorf("day is in the future")
	}
	return t, nil
}

func newFertilizerPrompt(plant *Plant) *inputPrompt {
	date := newDateInput("Date", "YYYY-MM-DD")
	input := newTextInput("Fertilizer Type", "liquid | granular")
	input.Validate = func(s string) error {
		if strings.HasPrefix(string(LiquidFertilizer), s) ||
			strings.HasPrefix(string(GranularFertilizer), s) {
			return nil
		}
		return fmt.Errorf("illegal input")
	}
	input.Focus()
	input.PromptStyle = focusedStyle
	input.TextStyle = focusedStyle
	return &inputPrompt{
		inputs:     []textinput.Model{date, input},
		focusIndex: 1,
		title:      "Add / Remove Fertilization Event",
		confirmAction: func(ip *inputPrompt) tea.Model {
			date, err := parseInputDate(ip.inputs[0].Value())
			if err != nil {
				// should already be verified by the Validate action on the input.
				panic(err)
			}

			var ft FertilizerType
			s := ip.inputs[1].Value()
			if strings.HasPrefix(string(LiquidFertilizer), s) {
				ft = LiquidFertilizer
			} else {
				ft = GranularFertilizer
			}

			plant.FertilizedAt = toggleEvent(plant.FertilizedAt, date)
			plant.FertilizedWith = &ft
			return nil
		},
	}
}

func newWateringPrompt(plant *Plant) *inputPrompt {
	date := newDateInput("Date", "YYYY-MM-DD")
	date.Focus()
	date.PromptStyle = focusedStyle
	date.TextStyle = focusedStyle
	return &inputPrompt{
		inputs: []textinput.Model{date},
		title:  "Add / Remove Watering Event",
		confirmAction: func(ip *inputPrompt) tea.Model {
			date, err := parseInputDate(ip.inputs[0].Value())
			if err != nil {
				// should already be verified by the Validate action on the input.
				panic(err)
			}

			plant.WateredAt = toggleEvent(plant.WateredAt, date)
			return nil
		},
	}
}

func newRepottingPrompt(plant *Plant) *inputPrompt {
	date := newDateInput("Date", "YYYY-MM-DD")
	newSize := newIntInput("New Pot Size", "in cm")
	newSize.Focus()
	newSize.PromptStyle = focusedStyle
	newSize.TextStyle = focusedStyle
	return &inputPrompt{
		inputs:     []textinput.Model{date, newSize},
		focusIndex: 1,
		title:      "Add / Remove Repotting Event",
		confirmAction: func(ip *inputPrompt) tea.Model {
			date, err := parseInputDate(ip.inputs[0].Value())
			if err != nil {
				// should already be verified by the Validate action on the input.
				panic(err)
			}

			newSize, _ := strconv.Atoi(ip.inputs[0].Value())
			plant.PotSize = newSize
			plant.RepottedAt = toggleEvent(plant.RepottedAt, date)
			return nil
		},
	}
}

func main() {
	pDB, err := readDBFile("/Users/ramon/.config/positive_hydration.json")
	if err != nil {
		fmt.Println("could not read DB file: ", err)
		os.Exit(2)
	}
	defer func() {
		if err := pDB.Close(); err != nil {
			fmt.Printf("Could not close DB: %v\n", err)
		}
	}()

	if _, err := tea.NewProgram(newShowPlants(pDB)).Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func readDBFile(location string) (*PlantDB, error) {
	pDB := &PlantDB{
		dbLocation: location,
	}
	data, err := os.ReadFile(location)
	if err != nil {
		if os.IsNotExist(err) {
			return pDB, nil
		}
		return nil, fmt.Errorf("could not read DB file: %w", err)
	}

	if err := json.Unmarshal(data, pDB); err != nil {
		return nil, fmt.Errorf("malformatted DB file: %w", err)
	}
	return pDB, nil
}

func (pDB *PlantDB) Close() error {
	pDB.normalise()

	data, err := json.Marshal(pDB)
	if err != nil {
		return fmt.Errorf("could not marshal plantDB to JSON: %w", err)
	}

	if err := os.WriteFile(pDB.dbLocation, data, 0600); err != nil {
		return fmt.Errorf("could not write DB file: %w", err)
	}

	return nil
}

func (pDB *PlantDB) normalise() {
	for _, plant := range pDB.Plants {
		sort.Slice(plant.WateredAt, func(i, j int) bool {
			return plant.WateredAt[i].Before(plant.WateredAt[j])
		})

		// makes sure there's only one watering-entry per day.
		var times []time.Time
		for _, watered := range plant.WateredAt {
			if len(times) > 0 &&
				times[len(times)-1].Year() == watered.Year() &&
				times[len(times)-1].YearDay() == watered.YearDay() {
				continue
			}
			times = append(times, watered)
		}
		plant.WateredAt = times
	}
}

type PlantDB struct {
	dbLocation string
	Plants     []*Plant `json:"plants"`
}

type Plant struct {
	Name                 string            `json:"name"`
	Variety              string            `json:"variety"`
	Location             string            `json:"location"`
	WateredAt            []time.Time       `json:"watered_at"`
	FertilizedAt         []time.Time       `json:"fertilized_at"`
	FertilizedWith       *FertilizerType   `json:"fertilizer_type,omitempty"`
	PotSize              int               `json:"pot_size"`
	RepottedAt           []time.Time       `json:"repotted_at"`
	WateringIntervals    SeasonalIntervals `json:"watering_intervals"`
	WetSoilDepth         int               `json:"wet_soil_depth"`
	FertilizingIntervals SeasonalIntervals `json:"fertilizing_intervals"`
	LightLevel           *LightLevel       `json:"light_level"`
	Comments             string            `json:"comments"`
	SourcedFrom          string            `json:"sourced_from"`
}

type FertilizerType string

const (
	LiquidFertilizer   FertilizerType = "liquid"
	GranularFertilizer FertilizerType = "granular"
)

type LightLevel string

var lightLevels = [...]LightLevel{
	"direct sunlight",
	"bright, indirect light",
	"bright / semi-shaded",
	"semi-shaded / shaded",
}

func (l *LightLevel) String() string {
	if l == nil {
		return ""
	}
	for i := range lightLevels {
		if lightLevels[i] == *l {
			return strconv.Itoa(i) + ": " + string(*l)
		}
	}
	return "unknown light level"
}

func (l *LightLevel) UnmarshalJSON(b []byte) error {
	var s int
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	if s < 0 || s > len(lightLevels) {
		return fmt.Errorf("invalid light level: %q", s)
	}

	*l = lightLevels[s-1]
	return nil
}
func (l *LightLevel) MarshalJSON() ([]byte, error) {
	if l == nil {
		return nil, fmt.Errorf("no light level to marshal")
	}

	var level int
	for i := range lightLevels {
		if lightLevels[i] == *l {
			level = i + 1
			break
		}
	}
	if level == 0 {
		return nil, fmt.Errorf("invalid light level %q", *l)
	}

	return json.Marshal(level)
}

func parseLightLevel(s string) (*LightLevel, error) {
	i, err := strconv.Atoi(s)
	if err != nil {
		for _, ll := range lightLevels {
			if string(ll) == s {
				return &ll, nil
			}
		}
		return nil, fmt.Errorf("invalid light level: %v", s)
	}
	if i > len(lightLevels)-1 {
		return nil, fmt.Errorf("light level out of range")
	}
	return &lightLevels[i], nil
}

type SeasonalIntervals struct {
	Summer int `json:"summer"`
	Winter int `json:"winter"`
}

func parseSeasonalIntervals(s string) (SeasonalIntervals, error) {
	splits := strings.Split(s, "/")
	if len(splits) > 2 {
		return SeasonalIntervals{}, fmt.Errorf("invalid seasonal interval")
	}
	if len(splits) == 0 {
		return SeasonalIntervals{}, fmt.Errorf("no interval given")
	}
	summer, err := strconv.Atoi(splits[0])
	if err != nil {
		return SeasonalIntervals{}, fmt.Errorf("invalid summer interval: %w", err)
	}

	if len(splits) == 1 {
		return SeasonalIntervals{
			Summer: summer,
			Winter: summer,
		}, nil
	}

	var winter int
	if splits[1] == "-" || splits[1] == "" {
		winter = 0
	} else {
		winter, err = strconv.Atoi(splits[1])
		if err != nil {
			return SeasonalIntervals{}, fmt.Errorf("invalid winter interval: %w", err)
		}
	}

	return SeasonalIntervals{
		Summer: summer,
		Winter: winter,
	}, nil
}

func (si SeasonalIntervals) String() string {
	var s strings.Builder
	s.WriteString(strconv.Itoa(si.Summer) + "/")
	if si.Winter != 0 {
		s.WriteString(strconv.Itoa(si.Winter))
	} else {
		s.WriteString("-")
	}
	return s.String()
}

func (p Plant) Render(includeStats bool) string {
	parts := []string{
		titleStyle.Render(p.Name),
		boxed.Render(p.Overview()),
		titleStyle.Render("Calendar Overview"),
		boxed.Render(calendar.NewRender(p.Events()...)),
	}
	if includeStats {
		parts = append(parts, p.renderStatistics())
	}
	return lipgloss.JoinVertical(lipgloss.Center, parts...)
}

func (p Plant) Events() []calendar.Event {
	var (
		wateredColor    = lipgloss.Color("#1d0ed1")
		fertilizedColor = lipgloss.Color("#004b26")
		repottedColor   = lipgloss.Color("#512013")
	)

	events := make(map[time.Time]lipgloss.Style)
	for _, w := range p.WateredAt {
		events[w.Truncate(24*time.Hour)] = lipgloss.NewStyle().Background(wateredColor)
	}
	for _, w := range p.FertilizedAt {
		t := w.Truncate(24 * time.Hour)
		if _, ok := events[t]; ok {
			events[t].Underline(true).Background(fertilizedColor)
		} else {
			events[t] = lipgloss.NewStyle().Background(fertilizedColor)

		}
	}
	for _, w := range p.RepottedAt {
		t := w.Truncate(24 * time.Hour)
		if _, ok := events[t]; ok {
			// Any fertilizing actions on the same day would not get displayed.
			// But please don't fertilize a plant that just got into fresh soil.
			events[t].Underline(true).Background(repottedColor)
		} else {
			events[t] = lipgloss.NewStyle().Background(repottedColor)
		}
	}

	e := make([]calendar.Event, 0, len(events))
	for time, style := range events {
		e = append(e, calendar.Event{Time: time, Style: style})
	}

	return e
}

func (p Plant) Overview() string {
	t1Rows := []table.Row{
		{"Variety", p.Variety},
		{"Location", p.Location},
		{"Last Watered", formatTimeInDays(last(p.WateredAt))},
		{"Last Fertilized", formatTimeInDays(last(p.FertilizedAt))},
	}

	t2Rows := []table.Row{
		{"Watering", p.WateringIntervals.String()},
		{"Fertilizing", p.FertilizingIntervals.String()},
		{"Light Level", p.LightLevel.String()},
		{"Soil Dryness", strconv.Itoa(p.WetSoilDepth) + "cm"},
	}

	additionalRows := []table.Row{}
	if lastRepot := last(p.RepottedAt); !lastRepot.IsZero() {
		additionalRows = append(additionalRows, table.Row{"Last Repotted", formatTimeInDays(lastRepot)})
	}
	if p.PotSize != 0 {
		additionalRows = append(additionalRows, table.Row{"Pot Size", p.formatPotSize()})
	}
	if p.FertilizedWith != nil {
		additionalRows = append(additionalRows, table.Row{"Fertilizer", string(*p.FertilizedWith)})
	}
	if p.SourcedFrom != "" {
		additionalRows = append(additionalRows, table.Row{"Sourced From", p.SourcedFrom})
	}

	for i, addR := range additionalRows {
		if i%2 == 0 {
			t1Rows = append(t1Rows, addR)
		} else {
			t2Rows = append(t2Rows, addR)
		}
	}

	styles := table.DefaultStyles()
	styles.Selected = styles.Cell.Padding(0)

	elements := []string{
		lipgloss.JoinHorizontal(lipgloss.Top,
			stripHeaderFromTable(table.New(
				table.WithColumns([]table.Column{
					{Title: "", Width: 17},
					{Title: "", Width: 30},
				}),
				table.WithRows(t1Rows),
				table.WithHeight(len(t1Rows)),
				table.WithStyles(styles),
			).View()),
			stripHeaderFromTable(table.New(
				table.WithColumns([]table.Column{
					{Title: "", Width: 13},
					{Title: "", Width: 22},
				}),
				table.WithRows(t2Rows),
				table.WithHeight(len(t2Rows)),
				table.WithStyles(styles),
			).View()),
		),
	}

	if p.Comments != "" {
		commentHeader := lipgloss.NewStyle().Width(17).Render("Comment")
		comment := lipgloss.NewStyle().Width(30 + 13 + 22).Render(p.Comments)
		elements = append(elements, commentHeader+comment)
	}

	return lipgloss.JoinVertical(lipgloss.Top, elements...)
}

func (p Plant) formatPotSize() string {
	if p.PotSize == 0 {
		return "unknown"
	}

	return strconv.Itoa(p.PotSize) + "cm"
}

func (p Plant) FilterValue() string {
	return p.Name + " " + p.Variety + " " + p.Location + " "
}
func (p Plant) Title() string {
	return p.Name
}
func (p Plant) Description() string {
	return "Location: " + p.Location + "\n" +
		"Last Watered: " + formatTimeInDays(last(p.WateredAt))
}

func toggleEvent(events []time.Time, newEvent time.Time) []time.Time {
	ny, nm, nd := newEvent.Date()
	for i, pastEvent := range events {
		py, pm, pd := pastEvent.Date()
		if ny == py && nm == pm && nd == pd {
			return append(events[:i], events[i+1:]...)
		}
	}

	return append(events, newEvent)
}

func (p Plant) renderStatistics() string {
	sort.Slice(p.WateredAt, func(i, j int) bool {
		return p.WateredAt[i].Before(p.WateredAt[j])
	})

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).Bold(false).Align(lipgloss.Left)
	s.Selected = s.Cell.Padding(0)

	t1Rows := []table.Row{
		{"Next Watering Day", p.nextScheduledWateringDay()},
		{"Last Watered", formatTimeInDays(last(p.WateredAt))},
		{"60 Days Avg Interval", formatAverage(average(p.WateredAt, 60))},
		{"Total Avg Interval", formatAverage(average(p.WateredAt, 0))},
	}

	t2Rows := []table.Row{
		{"Next Fertilizing Day", p.nextScheduledFertilizingDay()},
		{"Last Fertilized", formatTimeInDays(last(p.FertilizedAt))},
		{"90 Days Avg Interval", formatAverage(average(p.FertilizedAt, 90))},
		{"Total Avg Interval", formatAverage(average(p.FertilizedAt, 0))},
	}

	return boxed.Render(
		lipgloss.JoinHorizontal(lipgloss.Center,
			table.New(
				table.WithColumns([]table.Column{
					{Title: "Watering Stats", Width: 24},
					{Title: "", Width: 17},
				}),
				table.WithRows(t1Rows),
				table.WithHeight(len(t1Rows)),
				table.WithStyles(s),
			).View(),
			table.New(
				table.WithColumns([]table.Column{
					{Title: "Fertilizing Stats", Width: 24},
					{Title: "", Width: 17},
				}),
				table.WithRows(t2Rows),
				table.WithStyles(s),
				table.WithHeight(len(t2Rows)),
			).View(),
		))
}

// stripHeaderFromTable removes the Header from a table.Model.
func stripHeaderFromTable(table string) string {
	// Hack, but it works. It's important that the first line (or any, but the
	// first is easiest) has the required width.  This is encoded in the header,
	// which we strip away, so we need to add some padding back to the first
	// line.
	s := strings.SplitN(table, "\n", 3)
	header, firstLine, rest := s[0], s[1], s[2]
	// not sure if there's a nicer way to do this, but we create a format string
	// "%-3s", where 3 is the size of the header. This will add padding in the
	// form of spaces.
	return fmt.Sprintf(fmt.Sprintf("%%-%vs\n", lipgloss.Width(header)), firstLine) + rest
}

func (p Plant) nextScheduledWateringDay() string {
	//var last time.Time
	//if len(p.WateredAt) > 0 {
	//last = p.WateredAt[len(p.WateredAt)-1]
	//}

	if days, ok := scheduledIn(last(p.WateredAt), p.WateringIntervals); ok {
		return humanDaysDuration(days)
	}
	return "unknown"
}

func (p Plant) nextScheduledFertilizingDay() string {
	//var last time.Time
	//if len(p.FertilizedAt) > 0 {
	//last = p.FertilizedAt[len(p.FertilizedAt)-1]
	//}

	if days, ok := scheduledIn(last(p.FertilizedAt), p.FertilizingIntervals); ok {
		return humanDaysDuration(days)
	}
	return "unknown"
}

func scheduledIn(lastEvent time.Time, intervals SeasonalIntervals) (days int, ok bool) {
	now := time.Now()
	d := now.YearDay()

	// TODO: this is ugly, but at least it works.
	if d < 75 || d > 315 {
		// is Winter
		if intervals.Winter == 0 {
			if d < 75 {
				days = 75 - d
			} else {
				daysInYear := time.Date(now.Year(), 0, 0, 0, 0, 0, 0, time.UTC).YearDay()
				days = (daysInYear + 75) - d
			}

			// is Winter, calculate duration until summer
		} else {
			if lastEvent.IsZero() {
				return 0, false
			}

			next := lastEvent.Add(24 * time.Hour * time.Duration(intervals.Winter))
			days = daysFromToday(next)
		}
	} else {
		if lastEvent.IsZero() {
			return 0, false
		}

		// is Summer, and interval is always set.
		next := lastEvent.Add(24 * time.Hour * time.Duration(intervals.Summer))
		days = daysFromToday(next)
	}

	return days, true
}

func humanDaysDuration(days int) string {
	readableDuration := func(days int) string {
		if days < 14 {
			return strconv.Itoa(days) + " days"
		}

		weeks := days / 7
		remainder := days % 7
		if remainder > 3 {
			weeks++
		}
		var sign string
		if remainder != 0 {
			sign = "~"
		}
		return sign + strconv.Itoa(weeks) + " weeks"
	}

	switch days {
	case 0:
		return "today"
	case 1:
		return "tomorrow"
	case -1:
		return "yesterday"
	default:
		if days > 0 {
			return "in " + readableDuration(days)
		} else {
			return readableDuration(-days) + " ago"
		}
	}
}

// will probably be slightly off in case of daylight savings time, but so what.
func daysFromToday(t time.Time) int {
	today := time.Now().Truncate(24 * time.Hour)
	days := t.Truncate(24*time.Hour).Sub(today) / (24 * time.Hour)
	return int(days)
}

func formatTimeInDays(t time.Time) string {
	if t.IsZero() {
		return "never"
	}

	return humanDaysDuration(daysFromToday(t))
}

// assumes the times are ordered.
func last(times []time.Time) time.Time {
	if len(times) == 0 {
		return time.Time{}
	}
	return times[len(times)-1]
}

func formatAverage(avg float64) string {
	if math.IsNaN(avg) {
		return "n/a"
	}
	return strconv.FormatFloat(avg, 'f', 0, 64) + " days"
}

// average calculates the average of the duration between the given
// time points. If numLastDays is non-zero, it will only calculate the
// average of the durations between time points that lie within the
// last numLastDays.
func average(of []time.Time, numLastDays int) float64 {
	dur := time.Duration(numLastDays) * time.Hour * 24
	start := time.Now().Add(-dur)

	var intervals float64
	var numDataPoints = 0
	for i := 0; i < len(of)-1; i++ {
		if numLastDays != 0 && of[i].Before(start) {
			continue
		}
		intervals += of[i+1].Sub(of[i]).Hours() / 24
		numDataPoints++
	}
	if numDataPoints < 1 {
		return math.NaN()
	}

	return intervals / float64(numDataPoints)
}
