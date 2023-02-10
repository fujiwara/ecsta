package ecsta_test

import (
	"testing"
	"time"

	"github.com/Songmu/flextime"
	"github.com/fujiwara/ecsta"
)

func init() {
	flextime.Set(time.Date(2023, 2, 10, 11, 22, 33, 0, time.Local))
}

type logsOptionTest struct {
	title     string
	opt       *ecsta.LogsOption
	startTime time.Time
	endTime   time.Time
}

var logsOptionTests = []logsOptionTest{
	{
		title: "5 minutes ago from now",
		opt: &ecsta.LogsOption{
			Duration: 5 * time.Minute,
		},
		startTime: time.Date(2023, 2, 10, 11, 17, 33, 0, time.Local),
		endTime:   time.Date(2023, 2, 10, 11, 22, 33, 0, time.Local),
	},
	{
		title: "12 minutes ago from start time",
		opt: &ecsta.LogsOption{
			StartTime: "2023-01-12T22:33:44+09:00",
			Duration:  12 * time.Minute,
		},
		startTime: time.Date(2023, 1, 12, 22, 33, 44, 0, time.FixedZone("JST", 9*60*60)),
		endTime:   time.Date(2023, 1, 12, 22, 45, 44, 0, time.FixedZone("JST", 9*60*60)),
	},
	{
		opt: &ecsta.LogsOption{
			StartTime: "2023-01-02 11:22",
			Duration:  1 * time.Minute,
		},
		startTime: time.Date(2023, 1, 2, 11, 22, 0, 0, time.Local),
		endTime:   time.Date(2023, 1, 2, 11, 23, 0, 0, time.Local),
	},
}

func TestLogOptStartTime(t *testing.T) {
	for _, tt := range logsOptionTests {
		t.Run(tt.title, func(t *testing.T) {
			startTime, endTime, err := tt.opt.ResolveTimestamps()
			if err != nil {
				t.Errorf("failed to resolve start time: %s", err)
			}
			if startTime.Unix() != tt.startTime.Unix() {
				t.Errorf("startTime got %s, want %s", startTime, tt.startTime)
			}
			if endTime.Unix() != tt.endTime.Unix() {
				t.Errorf("endTime got %s, want %s", endTime, tt.endTime)
			}
		})
	}
}
