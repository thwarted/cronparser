package cronparser

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	regexSection    = regexp.MustCompile(`^[0-9*]+$`)
	regexRHS        = regexp.MustCompile(`^\d*$`)
	regexWhitespace = regexp.MustCompile(`[ \t]+`)
)

type CronParser struct {
	Environment map[string]string
	CronTab     []*CronEntry
}

type CronSection struct {
	Time     string
	Interval string
}

type CronEntry struct {
	Minute    *CronSection
	Hour      *CronSection
	Day       *CronSection
	Month     *CronSection
	DayOfWeek *CronSection
	User      string
	Command   string
}

func NewCronParser() *CronParser {
	return &CronParser{
		Environment: make(map[string]string),
		CronTab:     make([]*CronEntry, 0),
	}
}

func (cp *CronParser) ParseCronTab(body string) error {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimLeft(line, " \t")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if err := cp.ParseLine(line); err != nil {
			return err
		}
	}

	return nil
}

func (cp *CronParser) ParseLine(line string) error {
	if err := cp.ParseEntry(line); err == nil {
		return nil
	}

	return cp.ParseEnvironment(line)
}

func (cp *CronParser) ParseEnvironment(line string) error {
	key, value, err := parseEnvironment(line)
	if err != nil {
		return err
	}

	cp.Environment[key] = value
	return nil
}

func (cp *CronParser) ParseEntry(line string) error {
	ce, err := parseLine(line)
	if err != nil {
		return err
	}

	cp.CronTab = append(cp.CronTab, ce)
	return nil
}

func parseEnvironment(line string) (string, string, error) {
	parts := strings.SplitN(line, "=", 2)

	if parts[0] == "" {
		return "", "", fmt.Errorf("Could not locate key for environment line %q", line)
	}

	return parts[0], parts[1], nil
}

func parseSectionVar(cs **CronSection, str string) error {
	var err error
	*cs, err = parseSection(str)
	if err != nil {
		return fmt.Errorf("Minute is invalid: %q, error: %v", str, err)
	}
	return nil
}

func parseLine(line string) (*CronEntry, error) {
	strs := regexWhitespace.Split(line, 7)

	if len(strs) != 7 {
		return nil, fmt.Errorf("Not enough components found in cron line %q", line)
	}

	entry := &CronEntry{User: strs[5], Command: strs[6]}

	if err := parseSectionVar(&entry.Minute, strs[0]); err != nil {
		return nil, err
	}

	if err := parseSectionVar(&entry.Hour, strs[1]); err != nil {
		return nil, err
	}

	if err := parseSectionVar(&entry.Day, strs[2]); err != nil {
		return nil, err
	}

	if err := parseSectionVar(&entry.Month, strs[3]); err != nil {
		return nil, err
	}

	if err := parseSectionVar(&entry.DayOfWeek, strs[4]); err != nil {
		return nil, err
	}

	return entry, nil
}

func parseSection(section string) (*CronSection, error) {
	sections := strings.SplitN(section, "/", 2)

	strs := []string{}

	for _, sect := range sections {
		if !regexSection.MatchString(sect) {
			return nil, fmt.Errorf("Could not parse section part %q", sect)
		}

		strs = append(strs, sect)
	}

	switch len(strs) {
	case 2:
		break
	case 1:
		strs = append(strs, "")
	default:
		return nil, fmt.Errorf("Could not parser cron section %q", section)
	}

	if !regexRHS.MatchString(strs[1]) {
		return nil, fmt.Errorf("Right-hand side may not have a starred interval")
	}

	return &CronSection{Time: strs[0], Interval: strs[1]}, nil
}
