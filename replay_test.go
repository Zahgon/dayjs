package dayjs_test

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

// TestMain runs one recorded case and exits when DAYJS_REPLAY_CASE names it: the local zone is
// read from TZ once per process, so every case runs in a child process of its own.
func TestMain(m *testing.M) {
	if os.Getenv("DAYJS_REPLAY_CASE") != "" {
		replayChild()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

type replayCase struct {
	ID  string   `json:"id"`
	TZ  string   `json:"tz"`
	Now float64  `json:"now"`
	RC  int      `json:"rc"`
	Out []string `json:"out"`
}

// TestReplay checks every case of testdata/replay.jsonl, recorded from the original Day.js 1.11.23 on
// Node.js 24, line for line: parsing, local time across transitions, formatting in every locale,
// manipulation, differences, UTC offsets, time zones, durations, relative time and weeks.
func TestReplay(t *testing.T) {
	f, err := os.Open("testdata/replay.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var cases []replayCase
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		var c replayCase
		if err := json.Unmarshal(sc.Bytes(), &c); err != nil {
			t.Fatal(err)
		}
		cases = append(cases, c)
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("testdata/replay.jsonl holds no cases")
	}
	for _, c := range cases {
		t.Run(c.ID, func(t *testing.T) {
			t.Parallel()
			cmd := exec.Command(os.Args[0], "-test.run=^$")
			cmd.Env = append(os.Environ(), "TZ="+c.TZ, "DAYJS_NOW="+strconv.FormatFloat(c.Now, 'f', -1, 64),
				"DAYJS_REPLAY_CASE="+c.ID)
			stdout, err := cmd.Output()
			rc := 0
			if err != nil {
				exit, ok := err.(*exec.ExitError)
				if !ok {
					t.Fatal(err)
				}
				rc = exit.ExitCode()
			}
			if rc != c.RC {
				t.Fatalf("exit code %d, want %d", rc, c.RC)
			}
			got := strings.Split(strings.TrimSuffix(string(stdout), "\n"), "\n")
			if len(stdout) == 0 {
				got = nil
			}
			if len(got) != len(c.Out) {
				t.Fatalf("%d lines, want %d:\n%s", len(got), len(c.Out), stdout)
			}
			for i := range got {
				if got[i] != c.Out[i] {
					t.Errorf("line %d:\n got %s\nwant %s", i, got[i], c.Out[i])
				}
			}
		})
	}
}
