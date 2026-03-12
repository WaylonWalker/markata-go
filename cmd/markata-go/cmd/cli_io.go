package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

var currentCmd *cobra.Command

func activeCmd() *cobra.Command {
	return currentCmd
}

func outWriter() io.Writer {
	if cmd := activeCmd(); cmd != nil {
		return cmd.OutOrStdout()
	}
	return os.Stdout
}

func errWriter() io.Writer {
	if cmd := activeCmd(); cmd != nil {
		return cmd.ErrOrStderr()
	}
	return os.Stderr
}

func inReader() io.Reader {
	if cmd := activeCmd(); cmd != nil {
		return cmd.InOrStdin()
	}
	return os.Stdin
}

func outln(args ...any) {
	_, _ = fmt.Fprintln(outWriter(), args...)
}

func outlnf(format string, args ...any) {
	_, _ = fmt.Fprintf(outWriter(), format+"\n", args...)
}

func outText(text string) {
	_, _ = fmt.Fprint(outWriter(), text)
}

func out(format string, args ...any) {
	_, _ = fmt.Fprintf(outWriter(), format, args...)
}

func errln(args ...any) {
	_, _ = fmt.Fprintln(errWriter(), args...)
}

func errlnf(format string, args ...any) {
	_, _ = fmt.Fprintf(errWriter(), format+"\n", args...)
}

func errf(format string, args ...any) {
	_, _ = fmt.Fprintf(errWriter(), format, args...)
}

func infof(format string, args ...any) {
	if quiet {
		return
	}
	errlnf(format, args...)
}

func verbosef(format string, args ...any) {
	if !verbose || quiet {
		return
	}
	errlnf(format, args...)
}

func warnf(format string, args ...any) {
	if quiet {
		return
	}
	errlnf("Warning: "+format, args...)
}
