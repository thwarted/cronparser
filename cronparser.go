// Package cronparser is a small parser used to turn cron lines, and
// environment lines (therefore forming a crontab) into a set of data
// structures.
//
// Invoke cronparser like so:
//
//     package main
//
//     import (
//      "fmt"
//      "io/ioutil"
//      "os"
//     )
//
//     func main() {
//      cp := NewCronParser()
//      b, err := ioutil.ReadFile("/etc/crontab")
//      if err != nil {
//        os.Exit(1)
//      }
//
//      if err := cp.ParseCronTab(string(b)); err != nil {
//        os.Exit(1)
//      }
//
//      for key, value := range cp.Environment {
//        fmt.Println(key, value)
//      }
//
//      for _, entry := range cp.CronTab {
//        fmt.Println("Minute:", entry.Minute.Time, "/", entry.Minute.Interval)
//        // do the rest at your leisure.
//      }
//     }
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
	regexCmdStdin   = regexp.MustCompile(`([^\\])%`)
	regexCmdStdin2  = regexp.MustCompile(`\\%`)
)

// CronParser is a struct representing a full crontab and its environment.
// CronParser's functions typically modify the struct itself.
type CronParser struct {
	Environment map[string]string
	CronTab     []*CronEntry
}

// CronSection identifies a unit of time in cron, and an optional interval if
// Time is "*"
type CronSection struct {
	Time     string
	Interval string
}

// CronEntry Represents a full line in a crontab, with each portion of time
// delivered as CronSection structs. User and Command are also available in
// this struct.
type CronEntry struct {
	Minute    *CronSection
	Hour      *CronSection
	Day       *CronSection
	Month     *CronSection
	DayOfWeek *CronSection
	User      string
	Command   string
	Stdin     string
}

// NewCronParser constructs a new CronParser.
func NewCronParser() *CronParser {
	return &CronParser{
		Environment: make(map[string]string),
		CronTab:     make([]*CronEntry, 0),
	}
}

// ParseCronTab parses a whole body of text into indepedently sectioned
// Environment and CronTab sections inside CronParser. Each crontab is made of
// CronEntry structs which contain CronSection structs representing time
// values, and strings for the user and command. An error is returned if
// parsing fails.
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

// ParseLine parses a single crontab line into the CronParser. It self-selects
// whether or not it is an environment line or a cron specification. It first
// tries parsing a cron specification, and then tries to parse the environment,
// and returns errors if both fail. Please see ParseEnvironment and ParseEntry
// if you do not approve of this behavior.
func (cp *CronParser) ParseLine(line string) error {
	if err := cp.ParseEntry(line); err == nil {
		return nil
	}

	return cp.ParseEnvironment(line)
}

// ParseEnvironment parses an environment line into the CronParser.
// Environment lines are formed with a single key=value pair. They are
// case-sensitive, and are treated like environment variables throughout the
// code.  Note that this implementation does not support multiple key=value
// pairs on a single line.
func (cp *CronParser) ParseEnvironment(line string) error {
	key, value, err := parseEnvironment(line)
	if err != nil {
		return err
	}

	cp.Environment[key] = value
	return nil
}

// ParseEntry parses a single cron specification into the CronParser.
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

	// Handle this little known feature
	// Percent-signs (%) in the command, unless escaped with  backslash (\), will
	// be changed into newline characters, and all data after the first % will be
	// sent to the command as standard input.
	c := regexCmdStdin.ReplaceAllString(strs[6], "${1}\n")
	c = regexCmdStdin2.ReplaceAllString(c, "%")
	cs := strings.SplitN(c, "\n", 2)

	entry := &CronEntry{User: strs[5]}
	entry.Command = cs[0]
	if len(cs) > 1 {
		entry.Stdin = cs[1]
	}

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
