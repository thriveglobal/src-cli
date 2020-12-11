package api

import "flag"

// Flags encapsulates the standard flags that should be added to all commands
// that issue API requests.
type Flags struct {
	Dump    *bool
	GetCurl *bool
	Trace   *bool
}

// NewFlags instantiates a new Flags structure and attaches flags to the given
// flag set.
func NewFlags(flagSet *flag.FlagSet) *Flags {
	return &Flags{
		Dump:    flagSet.Bool("dump-requests", false, "Log GraphQL requests and responses to stdout"),
		GetCurl: flagSet.Bool("get-curl", false, "Print the curl command for executing this query and exit (WARNING: includes printing your access token!)"),
		Trace:   flagSet.Bool("trace", false, "Log the trace ID for requests. See https://docs.sourcegraph.com/admin/observability/tracing"),
	}
}

func DefaultFlags() *Flags {
	d := false
	return &Flags{
		Dump:    &d,
		GetCurl: &d,
		Trace:   &d,
	}
}
