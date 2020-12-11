package output

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/sourcegraph/src-cli/internal/output"
)

type CampaignOutput struct {
	*output.Output

	caps        output.Capabilities
	debugWriter io.Writer
	debugMu     sync.Mutex
}

var _ output.Writer = &CampaignOutput{}

type CampaignOutputOpts struct {
	output.OutputOpts
	DebugLog string
}

func NewCampaignOutput(w io.Writer, opts CampaignOutputOpts) *CampaignOutput {
	o := output.NewOutput(w, opts.OutputOpts)

	var debugWriter io.Writer
	if opts.DebugLog != "" {
		var err error
		debugWriter, err = os.Create(opts.DebugLog)
		if err != nil {
			o.WriteLine(output.Linef(output.EmojiWarning, output.StyleWarning, "Cannot open debug log %q: %v", opts.DebugLog, err))
		}
	}

	return &CampaignOutput{
		Output: o,
		caps: output.Capabilities{
			Color:  false,
			Isatty: false,
		},
		debugWriter: debugWriter,
	}
}

func (o *CampaignOutput) Verbose(s string) {
	// No check for the verbose flag here: if there's a debug log, we want this
	// to be written.
	o.debug(s)
	o.Output.Verbose(s)
}

func (o *CampaignOutput) Verbosef(format string, args ...interface{}) {
	o.debugf(format, args...)
	o.Output.Verbosef(format, args...)
}

func (o *CampaignOutput) VerboseLine(line output.FancyLine) {
	o.debugLine(line)
	o.Output.VerboseLine(line)
}

func (o *CampaignOutput) Write(s string) {
	o.debug(s)
	o.Output.Write(s)
}

func (o *CampaignOutput) Writef(format string, args ...interface{}) {
	o.debugf(format, args...)
	o.Output.Writef(format, args...)
}

func (o *CampaignOutput) WriteLine(line output.FancyLine) {
	o.debugLine(line)
	o.Output.WriteLine(line)
}

func (o *CampaignOutput) debug(s string) {
	if o.debugWriter != nil {
		o.debugMu.Lock()
		defer o.debugMu.Unlock()

		fmt.Fprintln(o.debugWriter, s)
	}
}

func (o *CampaignOutput) debugf(format string, args ...interface{}) {
	if o.debugWriter != nil {
		o.debugMu.Lock()
		defer o.debugMu.Unlock()

		fmt.Fprintf(o.debugWriter, format, args...)
		fmt.Fprint(o.debugWriter, "\n")
	}
}

func (o *CampaignOutput) debugLine(line output.FancyLine) {
	if o.debugWriter != nil {
		o.debugMu.Lock()
		defer o.debugMu.Unlock()

		line.Write(o.debugWriter, o.caps)
	}
}
