package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/WaylonWalker/markata-go/pkg/config"
	"github.com/WaylonWalker/markata-go/pkg/encryption"
	"github.com/WaylonWalker/markata-go/pkg/lifecycle"
	"github.com/WaylonWalker/markata-go/pkg/models"
	"github.com/WaylonWalker/markata-go/pkg/plugins"
	"github.com/spf13/cobra"
)

var (
	encryptionPasswordLength int
	encryptionCheckKey       string
	encryptionDryRun         bool
)

var encryptionCmd = &cobra.Command{
	Use:   "encryption",
	Short: "Utilities for encryption keys and passwords",
	Long: `Encryption utilities help you manage passwords and source-encrypted private posts.
`,
}

var generatePasswordCmd = &cobra.Command{
	Use:     "generate-password",
	Aliases: []string{"gen"},
	Short:   "Generate a policy-compliant encryption password",
	Long: `Generate a password that satisfies the default encryption policy (>=14 chars, strong entropy).

The password is printed to stdout only, making it easy to pipe into secret stores or copy it from your terminal.
`,
	Args: cobra.NoArgs,
	RunE: runGeneratePasswordCommand,
}

var checkPasswordCmd = &cobra.Command{
	Use:   "check",
	Short: "Check configured encryption key strength",
	Long: `Check configured encryption keys against the active policy.

By default this checks every key required by your config (default_key and private_tags mappings).
Use --key to check one specific key name.
`,
	Args: cobra.NoArgs,
	RunE: runCheckPasswordCommand,
}

var encryptPostsCmd = &cobra.Command{
	Use:   "encrypt-posts",
	Short: "Encrypt all private Markdown source bodies",
	Long: `Encrypt all private Markdown source bodies matched by the active content glob configuration.

Posts are encrypted in place by default. Draft, skipped, public, and already
source-encrypted posts are reported but not rewritten. Use --dry-run to preview
changes without modifying files.
`,
	Args: cobra.NoArgs,
	RunE: runEncryptPostsCommand,
}

func init() {
	encryptionCmd.AddCommand(generatePasswordCmd, checkPasswordCmd, encryptPostsCmd)
	generatePasswordCmd.Flags().IntVar(&encryptionPasswordLength, "length", encryption.DefaultMinPasswordLength, "password length (must be at least the configured minimum)")
	checkPasswordCmd.Flags().StringVar(&encryptionCheckKey, "key", "", "specific key name to check (default: all required keys)")
	encryptPostsCmd.Flags().BoolVar(&encryptionDryRun, "dry-run", false, "report files that would be encrypted without modifying them")
	rootCmd.AddCommand(encryptionCmd)
}

func runGeneratePasswordCommand(cmd *cobra.Command, _ []string) error {
	password, err := encryption.GeneratePassword(encryptionPasswordLength)
	if err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), password)
	return nil
}

func runCheckPasswordCommand(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	results, minDuration, minLength, err := evaluateEncryptionKeyPolicy(cfg, encryptionCheckKey)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No encryption keys configured to check.")
		return nil
	}

	failures := 0
	fmt.Fprintf(cmd.OutOrStdout(), "Policy: min_length=%d, min_estimated_crack_time=%s\n", minLength, formatCrackDurationHuman(minDuration))
	for _, result := range results {
		if result.Err != nil {
			failures++
			if result.Configured {
				fmt.Fprintf(cmd.OutOrStdout(), "FAIL %s (%s): %s (estimated=%s)\n", result.KeyName, result.EnvName, result.Err, formatCrackDurationHuman(result.Estimated))
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "FAIL %s (%s): %s\n", result.KeyName, result.EnvName, result.Err)
			}
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "PASS %s (%s): estimated=%s\n", result.KeyName, result.EnvName, formatCrackDurationHuman(result.Estimated))
	}

	if failures > 0 {
		return fmt.Errorf("%d encryption key(s) failed policy", failures)
	}

	if !cfg.Encryption.EnforceStrength {
		fmt.Fprintln(cmd.OutOrStdout(), "Warning: encryption.enforce_strength=false, builds will not enforce this policy.")
	}

	return nil
}

func runEncryptPostsCommand(cmd *cobra.Command, _ []string) error {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if !cfg.Encryption.Enabled {
		return fmt.Errorf("encryption is disabled in config")
	}

	results, minDuration, minLength, err := evaluateEncryptionKeyPolicy(cfg, "")
	if err != nil {
		return err
	}
	if err := failOnEncryptionKeyPolicyFailures(results, minDuration, minLength); err != nil {
		return err
	}

	files, err := encryptionContentFiles(cfg)
	if err != nil {
		return err
	}

	stats := encryptPostsStats{}
	candidates := make([]encryptPostResult, 0)
	for _, file := range files {
		result, err := encryptPostSourceFile(file, cfg, true)
		if err != nil {
			return err
		}
		stats.add(result)
		if result.Action == encryptPostActionEncrypted {
			candidates = append(candidates, result)
			if encryptionDryRun {
				fmt.Fprintf(cmd.OutOrStdout(), "WOULD ENCRYPT %s key=%s\n", result.Path, result.KeyName)
			}
		}
	}

	if !encryptionDryRun {
		for _, candidate := range candidates {
			result, err := encryptPostSourceFile(candidate.Path, cfg, false)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ENCRYPTED %s key=%s\n", result.Path, result.KeyName)
		}
	}

	action := "Encrypted"
	if encryptionDryRun {
		action = "Would encrypt"
	}
	fmt.Fprintf(cmd.OutOrStdout(), "%s %d private post(s); skipped %d already encrypted, %d public, %d draft/skip.\n",
		action, stats.Encrypted, stats.AlreadyEncrypted, stats.Public, stats.DraftOrSkip)

	return nil
}

func failOnEncryptionKeyPolicyFailures(results []encryptionKeyPolicyResult, minDuration time.Duration, minLength int) error {
	for _, result := range results {
		if result.Err != nil {
			return fmt.Errorf("encryption key %q failed policy (%s, min_length=%d, min_estimated_crack_time=%s): %w",
				result.KeyName, result.EnvName, minLength, formatCrackDurationHuman(minDuration), result.Err)
		}
	}
	return nil
}

func encryptionContentFiles(cfg *models.Config) ([]string, error) {
	lifecycleConfig := lifecycle.NewConfig()
	lifecycleConfig.ContentDir = "."
	lifecycleConfig.GlobPatterns = append([]string{}, cfg.GlobConfig.Patterns...)
	lifecycleConfig.Extra["use_gitignore"] = cfg.GlobConfig.UseGitignore

	manager := lifecycle.NewManager()
	manager.SetConfig(lifecycleConfig)

	globPlugin := plugins.NewGlobPlugin()
	if err := globPlugin.Configure(manager); err != nil {
		return nil, fmt.Errorf("configure content glob: %w", err)
	}
	if err := globPlugin.Glob(manager); err != nil {
		return nil, fmt.Errorf("scan content files: %w", err)
	}

	baseDir, err := filepath.Abs(lifecycleConfig.ContentDir)
	if err != nil {
		return nil, fmt.Errorf("resolve content dir: %w", err)
	}

	files := manager.Files()
	paths := make([]string, 0, len(files))
	for _, file := range files {
		if filepath.IsAbs(file) {
			paths = append(paths, file)
			continue
		}
		paths = append(paths, filepath.Join(baseDir, file))
	}
	return paths, nil
}

type encryptPostAction string

const (
	encryptPostActionEncrypted        encryptPostAction = "encrypted"
	encryptPostActionAlreadyEncrypted encryptPostAction = "already-encrypted"
	encryptPostActionPublic           encryptPostAction = "public"
	encryptPostActionDraftOrSkip      encryptPostAction = "draft-or-skip"
)

type encryptPostResult struct {
	Path    string
	KeyName string
	Action  encryptPostAction
}

type encryptPostsStats struct {
	Encrypted        int
	AlreadyEncrypted int
	Public           int
	DraftOrSkip      int
}

func (s *encryptPostsStats) add(result encryptPostResult) {
	switch result.Action {
	case encryptPostActionEncrypted:
		s.Encrypted++
	case encryptPostActionAlreadyEncrypted:
		s.AlreadyEncrypted++
	case encryptPostActionPublic:
		s.Public++
	case encryptPostActionDraftOrSkip:
		s.DraftOrSkip++
	}
}

func encryptPostSourceFile(path string, cfg *models.Config, dryRun bool) (encryptPostResult, error) {
	contentBytes, err := os.ReadFile(path)
	if err != nil {
		return encryptPostResult{}, fmt.Errorf("read %s: %w", path, err)
	}
	content := string(contentBytes)

	_, body, rawFrontmatter, err := plugins.ParseFrontmatterWithRaw(content)
	if err != nil {
		return encryptPostResult{}, fmt.Errorf("parse frontmatter %s: %w", path, err)
	}
	if encryption.IsSourceEncrypted(body) {
		return encryptPostResult{Path: path, Action: encryptPostActionAlreadyEncrypted}, nil
	}

	post, err := plugins.ParsePostFromContentWithConfig(path, content, cfg)
	if err != nil {
		return encryptPostResult{}, fmt.Errorf("parse %s: %w", path, err)
	}
	applyEncryptPostsPrivateTags(post, cfg)
	if post.Draft || post.Skip {
		return encryptPostResult{Path: path, Action: encryptPostActionDraftOrSkip}, nil
	}
	if !post.Private {
		return encryptPostResult{Path: path, Action: encryptPostActionPublic}, nil
	}

	keyName := strings.TrimSpace(post.SecretKey)
	if keyName == "" {
		keyName = strings.TrimSpace(cfg.Encryption.DefaultKey)
	}
	if keyName == "" {
		return encryptPostResult{}, fmt.Errorf("private post %s has no encryption key; set secret_key or encryption.default_key", path)
	}
	password := os.Getenv(plugins.EncryptionEnvPrefix + strings.ToUpper(keyName))
	if password == "" {
		return encryptPostResult{}, fmt.Errorf("private post %s requires %s%s", path, plugins.EncryptionEnvPrefix, strings.ToUpper(keyName))
	}
	if err := validateEncryptPostsPassword(password, cfg); err != nil {
		return encryptPostResult{}, fmt.Errorf("private post %s key %q failed policy: %w", path, keyName, err)
	}

	encryptedBody, err := encryption.EncryptSourceMarkdown(body, keyName, password)
	if err != nil {
		return encryptPostResult{}, fmt.Errorf("encrypt source body for %s: %w", path, err)
	}
	if dryRun {
		return encryptPostResult{Path: path, KeyName: keyName, Action: encryptPostActionEncrypted}, nil
	}

	nextContent := encryptedSourceDocument(rawFrontmatter, encryptedBody)
	mode := os.FileMode(0o600)
	if stat, err := os.Stat(path); err == nil {
		mode = stat.Mode().Perm()
	}
	if err := os.WriteFile(path, []byte(nextContent), mode); err != nil {
		return encryptPostResult{}, fmt.Errorf("write encrypted source %s: %w", path, err)
	}
	return encryptPostResult{Path: path, KeyName: keyName, Action: encryptPostActionEncrypted}, nil
}

func validateEncryptPostsPassword(password string, cfg *models.Config) error {
	minLength := cfg.Encryption.MinPasswordLength
	if minLength == 0 {
		minLength = encryption.DefaultMinPasswordLength
	}
	minDurationValue := cfg.Encryption.MinEstimatedCrackTime
	if minDurationValue == "" {
		minDurationValue = encryption.DefaultMinEstimatedCrackTime
	}
	minDuration, err := encryption.ParseEstimatedCrackDuration(minDurationValue)
	if err != nil {
		return fmt.Errorf("invalid encryption.min_estimated_crack_time: %w", err)
	}
	return encryption.ValidatePassword(password, minLength, minDuration)
}

func applyEncryptPostsPrivateTags(post *models.Post, cfg *models.Config) {
	if post == nil || cfg == nil || post.Skip || post.Draft {
		return
	}
	for _, tag := range post.Tags {
		if keyName, ok := cfg.Encryption.PrivateTags[strings.ToLower(tag)]; ok {
			post.Private = true
			if post.SecretKey == "" {
				post.SecretKey = keyName // pragma: allowlist secret
			}
			return
		}
	}
	if post.Template == "" {
		return
	}
	if keyName, ok := cfg.Encryption.PrivateTags[strings.ToLower(post.Template)]; ok {
		post.Private = true
		if post.SecretKey == "" {
			post.SecretKey = keyName // pragma: allowlist secret
		}
	}
}

func encryptedSourceDocument(rawFrontmatter, encryptedBody string) string {
	if rawFrontmatter == "" {
		return encryptedBody
	}
	return "---\n" + rawFrontmatter + "\n---\n" + encryptedBody
}

func formatCrackDurationHuman(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	if d < time.Second {
		return "<1s"
	}

	const year = 365 * 24 * time.Hour
	if d >= year {
		years := float64(d) / float64(year)
		if years >= 100 {
			return fmt.Sprintf("%.0fy", years)
		}
		return fmt.Sprintf("%.1fy", years)
	}

	const day = 24 * time.Hour
	if d >= day {
		days := float64(d) / float64(day)
		return fmt.Sprintf("%.1fd", days)
	}

	if d >= time.Hour {
		hours := float64(d) / float64(time.Hour)
		return fmt.Sprintf("%.1fh", hours)
	}

	return d.Round(time.Second).String()
}
