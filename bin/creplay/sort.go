package main

import (
	"regexp"
	"strconv"
)

var reNumber *regexp.Regexp

func init() {
	reNumber, _ = regexp.Compile("([0-9]*)$")
}

// ByNumber helper struct to sort by last number all the log files
type ByNumber []string

func (s ByNumber) Len() int {
	return len(s)
}
func (s ByNumber) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s ByNumber) Less(i, j int) bool {
	si := reNumber.FindString(s[i])
	sj := reNumber.FindString(s[j])
	di, _ := strconv.ParseInt(si, 10, 32)
	dj, _ := strconv.ParseInt(sj, 10, 32)
	return di < dj
}
