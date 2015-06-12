package cronparser

import (
	"encoding/json"
	"reflect"
	"testing"
)

func testTimeAndInterval(t *testing.T, section *CronSection, err error, expectedTime, expectedInterval string) {
	if err != nil {
		t.Fatal(err)
	}

	if section.Time != expectedTime {
		t.Fatalf("Time was not parsed correctly: %q", section.Time)
	}

	if section.Interval != expectedInterval {
		t.Fatalf("Interval was not parsed correctly, should be empty and is %q", section.Interval)
	}
}

var parseableLines = map[string]*CronEntry{
	"17 *    * * *   root    cd / && run-parts --report /etc/cron.hourly": &CronEntry{
		Minute:    &CronSection{Time: "17"},
		Hour:      &CronSection{Time: "*"},
		Day:       &CronSection{Time: "*"},
		Month:     &CronSection{Time: "*"},
		DayOfWeek: &CronSection{Time: "*"},
		User:      "root",
		Command:   "cd / && run-parts --report /etc/cron.hourly",
		Stdin:     "",
	},
	"25 6    * * *   root    test -x /usr/sbin/anacron || ( cd / && run-parts --report /etc/cron.daily )": &CronEntry{
		Minute:    &CronSection{Time: "25"},
		Hour:      &CronSection{Time: "6"},
		Day:       &CronSection{Time: "*"},
		Month:     &CronSection{Time: "*"},
		DayOfWeek: &CronSection{Time: "*"},
		User:      "root",
		Command:   "test -x /usr/sbin/anacron || ( cd / && run-parts --report /etc/cron.daily )",
		Stdin:     "",
	},
	"47 6    * * 7   fart test -x /usr/sbin/anacron || ( cd / && run-parts --report /etc/cron.weekly )": &CronEntry{
		Minute:    &CronSection{Time: "47"},
		Hour:      &CronSection{Time: "6"},
		Day:       &CronSection{Time: "*"},
		Month:     &CronSection{Time: "*"},
		DayOfWeek: &CronSection{Time: "7"},
		User:      "fart",
		Command:   "test -x /usr/sbin/anacron || ( cd / && run-parts --report /etc/cron.weekly )",
		Stdin:     "",
	},
}

var unparseableLines = []string{
	"#",
	"2",
	"*/* * * * * root /usr/local/rtm/bin/rtm 9 > /dev/null 2> /dev/null",
	"25/* 6    * * *   root    test -x /usr/sbin/anacron || ( cd / && run-parts --report /etc/cron.daily )",
}

func TestParseLine(t *testing.T) {
	for line, cronentry := range parseableLines {
		tstCe, err := parseLine(line)
		if err != nil {
			t.Fatal("Could not parse line:", line)
		}

		if !reflect.DeepEqual(cronentry, tstCe) {
			t.Fatal("Cron entries are not equal for line:", line)
		}
	}

	for _, line := range unparseableLines {
		_, err := parseLine(line)
		if err == nil {
			t.Fatal("Parsing succeeded for bad line:", line)
		}
	}
}

func TestParseSection(t *testing.T) {
	section, err := parseSection("1")
	testTimeAndInterval(t, section, err, "1", "")

	section, err = parseSection("*/1")
	testTimeAndInterval(t, section, err, "*", "1")

	section, err = parseSection("*-1")
	if err == nil {
		t.Fatal("did not error on invalid input")
	}

	section, err = parseSection("d")
	if err == nil {
		t.Fatal("did not error on invalid input")
	}

	section, err = parseSection("1/*")
	if err == nil {
		t.Fatal("did not error on invalid input")
	}
}

func testParseEnvironmentHelper(t *testing.T, line, expectedKey, expectedValue string) {
	key, value, err := parseEnvironment(line)
	if err != nil {
		t.Fatal(err)
	}

	if key != expectedKey || value != expectedValue {
		t.Fatal("Could not retrieve expected values after parseEnvironment")
	}
}

func TestParseEnvironment(t *testing.T) {
	testParseEnvironmentHelper(t, "foo=bar", "foo", "bar")
	testParseEnvironmentHelper(t, "foo=", "foo", "")
	testParseEnvironmentHelper(t, "FOO=bar", "FOO", "bar")

	if _, _, err := parseEnvironment("="); err == nil {
		t.Fatal("Did not error on '='")
	}

	if _, _, err := parseEnvironment(""); err == nil {
		t.Fatal("Did not error on ''")
	}
}

func TestCronParser(t *testing.T) {
	cp := NewCronParser()

	for line, cronentry := range parseableLines {
		if err := cp.ParseLine(line); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(cronentry, cp.CronTab[len(cp.CronTab)-1]) {
			t.Fatal("Structures parsed do not equal expectations")
		}

		if err := cp.ParseEntry(line); err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(cronentry, cp.CronTab[len(cp.CronTab)-1]) {
			t.Fatal("Structures parsed do not equal expectations")
		}
	}

	if err := cp.ParseEnvironment("foo=bar"); err != nil {
		t.Fatal(err)
	}

	if err := cp.ParseEnvironment("bar=baz"); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(map[string]string{"foo": "bar", "bar": "baz"}, cp.Environment) {
		t.Fatal("Expectation did not equal parsed environment")
	}

	if err := cp.ParseEnvironment("="); err == nil {
		t.Fatal("Parsed '=' when should have failed")
	}

	if !reflect.DeepEqual(map[string]string{"foo": "bar", "bar": "baz"}, cp.Environment) {
		t.Fatal("Parsing bad environment line yielded dirty environment")
	}
}

func TestCronParserParseCronTab(t *testing.T) {
	// FIXME replace this with a dir of crontabs to parse
	crontab := `
# /etc/crontab: system-wide crontab
# Unlike any other crontab you don't have to run the crontab
# command to install the new version when you edit this file
# and files in /etc/cron.d. These files also have username fields,
# that none of the other crontabs do.

SHELL=/bin/sh
PATH=/usr/local/sbin:/usr/local/bin:/sbin:/bin:/usr/sbin:/usr/bin

# m h dom mon dow user  command
17 *    * * *   root    cd / && run-parts --report /etc/cron.hourly
25 6    * * *   root    test -x /usr/sbin/anacron || ( cd / && run-parts --report /etc/cron.daily )
47 6    * * 7   root    test -x /usr/sbin/anacron || ( cd / && run-parts --report /etc/cron.weekly )
52 6    1 * *   root    test -x /usr/sbin/anacron || ( cd / && run-parts --report /etc/cron.monthly )
#
*/1 * * * * root /usr/local/rtm/bin/rtm 9 > /dev/null 2> /dev/null
*/2 * * * * root somecommand >> /tmp/commandlog.$( date +\%Y\%m\%d )
*/15 * * * * root somecommand >> /tmp/commandlog.$( date +\%Y\%m\%d ) %command has trailing space% stdin line with leading space
`
	cp := NewCronParser()

	if err := cp.ParseCronTab(crontab); err != nil {
		t.Fatal(err)
	}

	structtab := &CronParser{
		Environment: map[string]string{
			"SHELL": "/bin/sh",
			"PATH":  "/usr/local/sbin:/usr/local/bin:/sbin:/bin:/usr/sbin:/usr/bin",
		},
		CronTab: []*CronEntry{
			&CronEntry{
				Minute:    &CronSection{Time: "17"},
				Hour:      &CronSection{Time: "*"},
				Day:       &CronSection{Time: "*"},
				Month:     &CronSection{Time: "*"},
				DayOfWeek: &CronSection{Time: "*"},
				User:      "root",
				Command:   "cd / && run-parts --report /etc/cron.hourly",
				Stdin:     "",
			},
			&CronEntry{
				Minute:    &CronSection{Time: "25"},
				Hour:      &CronSection{Time: "6"},
				Day:       &CronSection{Time: "*"},
				Month:     &CronSection{Time: "*"},
				DayOfWeek: &CronSection{Time: "*"},
				User:      "root",
				Command:   "test -x /usr/sbin/anacron || ( cd / && run-parts --report /etc/cron.daily )",
				Stdin:     "",
			},
			&CronEntry{
				Minute:    &CronSection{Time: "47"},
				Hour:      &CronSection{Time: "6"},
				Day:       &CronSection{Time: "*"},
				Month:     &CronSection{Time: "*"},
				DayOfWeek: &CronSection{Time: "7"},
				User:      "root",
				Command:   "test -x /usr/sbin/anacron || ( cd / && run-parts --report /etc/cron.weekly )",
				Stdin:     "",
			},
			&CronEntry{
				Minute:    &CronSection{Time: "52"},
				Hour:      &CronSection{Time: "6"},
				Day:       &CronSection{Time: "1"},
				Month:     &CronSection{Time: "*"},
				DayOfWeek: &CronSection{Time: "*"},
				User:      "root",
				Command:   "test -x /usr/sbin/anacron || ( cd / && run-parts --report /etc/cron.monthly )",
				Stdin:     "",
			},
			&CronEntry{
				Minute:    &CronSection{Time: "*", Interval: "1"},
				Hour:      &CronSection{Time: "*"},
				Day:       &CronSection{Time: "*"},
				Month:     &CronSection{Time: "*"},
				DayOfWeek: &CronSection{Time: "*"},
				User:      "root",
				Command:   "/usr/local/rtm/bin/rtm 9 > /dev/null 2> /dev/null",
				Stdin:     "",
			},
			&CronEntry{
				Minute:    &CronSection{Time: "*", Interval: "2"},
				Hour:      &CronSection{Time: "*"},
				Day:       &CronSection{Time: "*"},
				Month:     &CronSection{Time: "*"},
				DayOfWeek: &CronSection{Time: "*"},
				User:      "root",
				Command:   `somecommand >> /tmp/commandlog.$( date +%Y%m%d )`,
				Stdin:     "",
			},
			&CronEntry{
				Minute:    &CronSection{Time: "*", Interval: "15"},
				Hour:      &CronSection{Time: "*"},
				Day:       &CronSection{Time: "*"},
				Month:     &CronSection{Time: "*"},
				DayOfWeek: &CronSection{Time: "*"},
				User:      "root",
				Command:   `somecommand >> /tmp/commandlog.$( date +%Y%m%d ) `,
				Stdin:     "command has trailing space\n stdin line with leading space",
			},
		},
	}

	if !reflect.DeepEqual(structtab, cp) {
		content, _ := json.MarshalIndent(cp, "", "  ")
		t.Log("JSON Dumping crontab:", string(content))
		t.Fatalf("Crontab was not parsed properly")
	}
}
