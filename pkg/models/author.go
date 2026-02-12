package models

import (
	"fmt"
	"strings"
)

// RoleAuthor is the simple role name for a primary author.
const RoleAuthor = "author"

// Author represents an author or contributor to content
type Author struct {
	ID      string            `json:"id" yaml:"id" toml:"id"`
	Name    string            `json:"name" yaml:"name" toml:"name"`
	Bio     *string           `json:"bio,omitempty" yaml:"bio,omitempty" toml:"bio,omitempty"`
	Email   *string           `json:"email,omitempty" yaml:"email,omitempty" toml:"email,omitempty"`
	Avatar  *string           `json:"avatar,omitempty" yaml:"avatar,omitempty" toml:"avatar,omitempty"`
	URL     *string           `json:"url,omitempty" yaml:"url,omitempty" toml:"url,omitempty"`
	Social  map[string]string `json:"social,omitempty" yaml:"social,omitempty" toml:"social,omitempty"`
	Guest   bool              `json:"guest,omitempty" yaml:"guest,omitempty" toml:"guest,omitempty"`
	Active  bool              `json:"active,omitempty" yaml:"active,omitempty" toml:"active,omitempty"`
	Default bool              `json:"default,omitempty" yaml:"default,omitempty" toml:"default,omitempty"`

	// Level 1: CReDiT academic roles
	Contributions []string `json:"contributions,omitempty" yaml:"contributions,omitempty" toml:"contributions,omitempty"`

	// Level 2: Simple role system
	Role *string `json:"role,omitempty" yaml:"role,omitempty" toml:"role,omitempty"`

	// Level 3: Custom free-form contribution
	Contribution *string `json:"contribution,omitempty" yaml:"contribution,omitempty" toml:"contribution,omitempty"`

	// Details is an optional per-post description of what the author did.
	// Typically set via frontmatter overrides, displayed as a tooltip on hover.
	Details *string `json:"details,omitempty" yaml:"details,omitempty" toml:"details,omitempty"`
}

// CReDiTRoles defines standard CReDiT contributor roles taxonomy
// ANSI/NISO standard Z39.104-2022
var CReDiTRoles = []string{
	"conceptualization",      // Ideas and research questions
	"data-curation",          // Data management and cleaning
	"formal-analysis",        // Statistical analysis and interpretation
	"funding-acquisition",    // Securing financial support
	"investigation",          // Research and data collection
	"methodology",            // Study design and methods
	"project-administration", // Project management and coordination
	"resources",              // Materials, reagents, and tools
	"software",               // Programming and software development
	"supervision",            // Mentorship and oversight
	"validation",             // Verification and reproducibility
	"visualization",          // Data visualization and presentation
	"writing-original-draft", // Initial manuscript preparation
	"writing-review-editing", // Revision and editing
}

// SimpleRoles defines common roles for blog/content sites
var SimpleRoles = []string{
	RoleAuthor,    // Primary writer
	"editor",      // Editorial contributions
	"designer",    // Visual/UX design
	"maintainer",  // Project maintenance
	"contributor", // Specific contributions
	"reviewer",    // Content review
	"translator",  // Translation work
}

// ValidateCReDiTContributions checks if all contributions are valid CReDiT roles
func (a *Author) ValidateCReDiTContributions() error {
	for _, contribution := range a.Contributions {
		if !isValidCReDiTRole(contribution) {
			return fmt.Errorf("invalid CReDiT contribution: %s, valid roles are: %s",
				contribution, strings.Join(CReDiTRoles, ", "))
		}
	}
	return nil
}

// ValidateSimpleRole checks if role is in simple roles list
func (a *Author) ValidateSimpleRole() error {
	if a.Role == nil {
		return nil // Role is optional
	}
	if !isValidSimpleRole(*a.Role) {
		return fmt.Errorf("invalid role: %s, valid roles are: %s",
			*a.Role, strings.Join(SimpleRoles, ", "))
	}
	return nil
}

// HasContribution checks if author has a specific contribution type
func (a *Author) HasContribution(contribution string) bool {
	for _, c := range a.Contributions {
		if c == contribution {
			return true
		}
	}
	return false
}

// GetRoleDisplay returns a human-readable display of the author's role/contribution
func (a *Author) GetRoleDisplay() string {
	if a.Contribution != nil && *a.Contribution != "" {
		return *a.Contribution
	}
	if a.Role != nil && *a.Role != "" {
		return *a.Role
	}
	if len(a.Contributions) > 0 {
		return strings.Join(a.Contributions, ", ")
	}
	return "Author"
}

// IsPrimaryContributor checks if author has writing/conceptualization roles
func (a *Author) IsPrimaryContributor() bool {
	// Check CReDiT contributions for primary author roles
	primaryRoles := map[string]bool{
		"conceptualization":      true,
		"methodology":            true,
		"writing-original-draft": true,
		"investigation":          true,
	}

	for _, contribution := range a.Contributions {
		if primaryRoles[contribution] {
			return true
		}
	}

	// Check simple role
	if a.Role != nil && *a.Role == RoleAuthor {
		return true
	}

	return false
}

// ValidateAuthor validates author data according to the specified level
func (a *Author) Validate() error {
	if a.ID == "" {
		return fmt.Errorf("author ID is required")
	}
	if a.Name == "" {
		return fmt.Errorf("author name is required")
	}

	// Validate based on what fields are populated
	if err := a.ValidateCReDiTContributions(); err != nil {
		return err
	}

	return a.ValidateSimpleRole()
}

// ValidateAuthors validates a collection of authors and enforces business rules
func ValidateAuthors(authors map[string]Author) error {
	if authors == nil {
		return nil
	}

	defaultCount := 0
	for id := range authors {
		author := authors[id]
		if err := author.Validate(); err != nil {
			return fmt.Errorf("author %s: %w", id, err)
		}

		if author.Default {
			defaultCount++
			if defaultCount > 1 {
				return fmt.Errorf("only one author can be marked as default, found %d", defaultCount)
			}
		}
	}

	return nil
}

// GetDefaultAuthor returns the author marked as default
func GetDefaultAuthor(authors map[string]Author) (defaultAuthor *Author, defaultID string) {
	if authors == nil {
		return nil, ""
	}

	for id := range authors {
		if authors[id].Default {
			a := authors[id]
			defaultAuthor = &a
			defaultID = id
			return
		}
	}

	// Fallback: return first active author if no default specified
	for id := range authors {
		if authors[id].Active {
			a := authors[id]
			defaultAuthor = &a
			defaultID = id
			return
		}
	}

	return nil, ""
}

// Helper functions
func isValidCReDiTRole(role string) bool {
	for _, validRole := range CReDiTRoles {
		if role == validRole {
			return true
		}
	}
	return false
}

func isValidSimpleRole(role string) bool {
	for _, validRole := range SimpleRoles {
		if role == validRole {
			return true
		}
	}
	return false
}
