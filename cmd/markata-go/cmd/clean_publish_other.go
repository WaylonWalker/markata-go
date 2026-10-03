//go:build !linux

package cmd

func exchangeOutputDirectories(_, _ string) error {
	return errAtomicExchangeUnsupported
}
