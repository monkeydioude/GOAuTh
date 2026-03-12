package timed

import (
	"time"

	"golang.org/x/exp/constraints"
)

const (
	Day  = 24 * time.Hour
	Week = 7 * Day
)

func Seconds[IntT constraints.Integer](seconds IntT) time.Duration {
	return time.Duration(seconds) * time.Second
}

func Minutes[IntT constraints.Integer](minutes IntT) time.Duration {
	return time.Duration(minutes) * time.Minute
}

func Hours[IntT constraints.Integer](hours IntT) time.Duration {
	return time.Duration(hours) * time.Hour
}

func Days[IntT constraints.Integer](days IntT) time.Duration {
	return time.Duration(days) * Day
}

func Weeks[IntT constraints.Integer](weeks IntT) time.Duration {
	return time.Duration(weeks) * Week
}
