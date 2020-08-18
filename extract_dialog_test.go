package main

import (
	"reflect"
	"testing"
	"time"
)

func duration(d string) time.Duration {
	dur, err := time.ParseDuration(d)
	if err != nil {
		panic(err)
	}
	return dur
}

func Test_combineIntervals(t *testing.T) {
	type args struct {
		intervals []Interval
	}
	tests := []struct {
		name string
		args args
		want []Interval
	}{
		{
			name: "NonOverlapping",
			args: args{
				intervals: []Interval{
					Interval{start: "00:00:00.001", end: "00:00:00.005"},
					Interval{start: "00:00:00.008", end: "00:00:00.009"},
				},
			},
			want: []Interval{
				Interval{start: "00:00:00.001", end: "00:00:00.005"},
				Interval{start: "00:00:00.008", end: "00:00:00.009"},
			},
		},
		{
			name: "Backwards",
			args: args{
				intervals: []Interval{
					Interval{start: "00:00:00.008", end: "00:00:00.009"},
					Interval{start: "00:00:00.001", end: "00:00:00.005"},
				},
			},
			want: []Interval{
				Interval{start: "00:00:00.001", end: "00:00:00.005"},
				Interval{start: "00:00:00.008", end: "00:00:00.009"},
			},
		},
		{
			name: "WithinThreshold",
			args: args{
				intervals: []Interval{
					Interval{start: "00:00:00.001", end: "00:00:00.006"},
					Interval{start: "00:00:00.008", end: "00:00:00.009"},
				},
			},
			want: []Interval{
				Interval{start: "00:00:00.001", end: "00:00:00.009"},
			},
		},
		{
			name: "Contained",
			args: args{
				intervals: []Interval{
					Interval{start: "00:00:00.001", end: "00:00:00.005"},
					Interval{start: "00:00:00.003", end: "00:00:00.004"},
					Interval{start: "00:00:00.004", end: "00:00:00.005"},
					Interval{start: "00:00:00.005", end: "00:00:00.009"},
					Interval{start: "00:00:01.005", end: "00:00:03.009"},
				},
			},
			want: []Interval{
				Interval{start: "00:00:00.001", end: "00:00:00.009"},
				Interval{start: "00:00:01.005", end: "00:00:03.009"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := combineIntervals(tt.args.intervals, duration("0.002s")); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("combineIntervals() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isGapOverThreshold(t *testing.T) {
	type args struct {
		start        string
		end          string
		gapThreshold time.Duration
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "WithinThreshold",
			args: args{
				start:        "00:00:00.001",
				end:          "00:00:00.002",
				gapThreshold: duration("0.002s"),
			},
			want: false,
		},
		{
			name: "WithinThreshold_Backwards",
			args: args{
				start:        "00:00:00.002",
				end:          "00:00:00.001",
				gapThreshold: duration("0.002s"),
			},
			want: false,
		},
		{
			name: "OverThreshold",
			args: args{
				start:        "00:00:00.001",
				end:          "00:00:00.005",
				gapThreshold: duration("0.002s"),
			},
			want: true,
		},
		{
			name: "OverThreshold_Backwards",
			args: args{
				start:        "00:00:00.005",
				end:          "00:00:00.001",
				gapThreshold: duration("0.002s"),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isGapOverThreshold(tt.args.start, tt.args.end, tt.args.gapThreshold); got != tt.want {
				t.Errorf("gapOverThreshold() = %v, want %v", got, tt.want)
			}
		})
	}
}
