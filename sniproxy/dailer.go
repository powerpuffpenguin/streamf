package sniproxy

import (
	"regexp"
	"strings"
	"time"

	"github.com/powerpuffpenguin/streamf/dialer"
)

type accuracyMatcher struct {
	dialer   dialer.Dialer
	duration time.Duration
}
type orderMatcher struct {
	dialer   dialer.Dialer
	duration time.Duration

	value  string
	prefix bool
}

func (o *orderMatcher) Match(s string) bool {
	if o.prefix {
		return strings.HasPrefix(s, o.value)
	}
	return strings.HasSuffix(s, o.value)
}

type regexpMatcher struct {
	dialer   dialer.Dialer
	duration time.Duration

	value *regexp.Regexp
}

func (o *regexpMatcher) Match(s string) bool {
	return o.value.MatchString(s)
}
