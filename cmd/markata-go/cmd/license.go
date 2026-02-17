package cmd

import (
	"fmt"

	"github.com/WaylonWalker/markata-go/pkg/models"
)

type licenseChoice struct {
	Key         string
	Label       string
	Description string
}

func licenseChoices() []licenseChoice {
	choices := make([]licenseChoice, 0, len(models.LicenseOptions)+1)
	for _, opt := range models.LicenseOptions {
		label := opt.Name
		if opt.Recommended {
			label = fmt.Sprintf("%s (recommended)", label)
		}
		desc := opt.Description
		if opt.URL != "" {
			desc = fmt.Sprintf("%s %s", desc, opt.URL)
		}
		choices = append(choices, licenseChoice{
			Key:         opt.Key,
			Label:       label,
			Description: desc,
		})
	}
	choices = append(choices, licenseChoice{
		Key:         "false",
		Label:       "No license (opt out)",
		Description: "Hide the license footer and skip the warning",
	})
	return choices
}

func defaultLicenseIndex(choices []licenseChoice) int {
	for i, choice := range choices {
		if choice.Key == models.DefaultLicenseKey {
			return i
		}
	}
	return 0
}

func licenseDisplayStrings(choices []licenseChoice) []string {
	displays := make([]string, len(choices))
	for i, choice := range choices {
		displays[i] = fmt.Sprintf("%s – %s", choice.Label, choice.Description)
	}
	return displays
}
