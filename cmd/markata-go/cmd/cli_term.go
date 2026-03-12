package cmd

import (
	"os"
	"strings"
)

func inputIsTerminal() bool {
	return streamIsTerminal(inReader())
}

func outputIsTerminal() bool {
	return streamIsTerminal(outWriter())
}

func streamIsTerminal(stream any) bool {
	return fileLikeTerminal(stream)
}

func fileLikeTerminal(stream any) bool {
	file, ok := stream.(*os.File)
	if !ok {
		return false
	}
	stat, err := file.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

func colorEnabledOnOutput() bool {
	return colorEnabledFor(outputIsTerminal())
}

func colorEnabledFor(isTTY bool) bool {
	if noColor || !isTTY {
		return false
	}
	if os.Getenv("TERM") == "dumb" {
		return false
	}
	return strings.TrimSpace(os.Getenv("NO_COLOR")) == ""
}
