package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/orot-dev/orot-kit/internal/builder"
	"github.com/orot-dev/orot-kit/internal/output"
	kitsecret "github.com/orot-dev/orot-kit/internal/secret"
	"github.com/spf13/cobra"
)

type secretOptions struct {
	length  int
	copy    bool
	noPrint bool
	envKey  string
	format  string
	symbols bool
	prefix  string
}

func registerSecretCommands(root *cobra.Command) {
	secretCmd := &cobra.Command{
		Use:   "secret",
		Short: "Generate secure random secrets",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSecretBuilder(cmd)
		},
	}
	secretCmd.AddCommand(newSecretSubcommand("password", "Generate random password", 32, "text"))
	secretCmd.AddCommand(newSecretSubcommand("token", "Generate random token", 32, "text"))
	secretCmd.AddCommand(newSecretSubcommand("api-key", "Generate prefixed API key", 32, "text"))
	secretCmd.AddCommand(newSecretSubcommand("jwt", "Generate JWT secret", 32, "hex"))
	secretCmd.AddCommand(newSecretSubcommand("hex", "Generate hex key", 32, "hex"))
	secretCmd.AddCommand(newSecretSubcommand("base64", "Generate base64 key", 32, "base64"))
	secretCmd.AddCommand(newSecretSubcommand("env", "Generate .env KEY=value secret", 32, "base64"))
	secretCmd.AddCommand(newSecretUUIDCommand())
	root.AddCommand(secretCmd)
}

func newSecretUUIDCommand() *cobra.Command {
	options := secretOptions{}
	command := &cobra.Command{
		Use:   "uuid",
		Short: "Generate UUID v4",
		RunE: func(cmd *cobra.Command, args []string) error {
			value, err := kitsecret.UUID()
			if err != nil {
				return err
			}
			summary := "Generated with crypto/rand. Secret is not stored by kit."
			if options.copy {
				if err := kitsecret.CopyToClipboard(context.Background(), value); err != nil {
					summary += "\nClipboard copy failed: " + err.Error()
				} else {
					summary += "\nCopied to clipboard."
				}
			}
			result := value
			if options.noPrint {
				result = "(hidden)"
			}
			return writer(cmd).Write(output.Result{
				Title:   "Secret UUID",
				Command: []string{uuidCommandPreview(options)},
				Result:  result,
				Summary: summary,
			})
		},
	}
	command.Flags().BoolVar(&options.copy, "copy", false, "copy generated UUID to clipboard")
	command.Flags().BoolVar(&options.noPrint, "no-print", false, "do not print generated UUID")
	return command
}

func runSecretBuilder(cmd *cobra.Command) error {
	prompt := builder.NewPrompt(os.Stdin, cmd.OutOrStdout())
	kind, err := prompt.Select("Secret type", []builder.Choice{
		{Label: "password", Value: "password"},
		{Label: "token", Value: "token"},
		{Label: "api key", Value: "api-key"},
		{Label: "jwt", Value: "jwt"},
		{Label: "hex", Value: "hex"},
		{Label: "base64", Value: "base64"},
		{Label: ".env line", Value: "env"},
		{Label: "uuid", Value: "uuid"},
	}, 1)
	if err != nil {
		return err
	}
	if kind == "uuid" {
		value, err := kitsecret.UUID()
		if err != nil {
			return err
		}
		return writer(cmd).Write(output.Result{
			Title:   "Secret UUID",
			Command: []string{"kit internal secret uuid"},
			Summary: "Generated with crypto/rand.",
			Result:  value,
		})
	}
	lengthValue, err := prompt.Ask("Length", "32")
	if err != nil {
		return err
	}
	length, err := strconv.Atoi(lengthValue)
	if err != nil {
		return err
	}
	options := secretOptions{length: length, envKey: "SECRET", format: defaultSecretFormat(kind), symbols: true, prefix: "key"}
	if kind == "env" {
		options.envKey, err = prompt.Ask(".env key", "SECRET")
		if err != nil {
			return err
		}
	}
	if kind == "jwt" {
		options.format, err = prompt.Select("Output format", []builder.Choice{
			{Label: "hex", Value: "hex"},
			{Label: "base64", Value: "base64"},
			{Label: ".env line", Value: "env"},
		}, 0)
		if err != nil {
			return err
		}
		if options.format == "env" {
			options.envKey, err = prompt.Ask(".env key", "JWT_SECRET")
			if err != nil {
				return err
			}
		}
	}
	if kind == "password" {
		options.symbols, err = prompt.Confirm("Include symbols?", true)
		if err != nil {
			return err
		}
	}
	if kind == "api-key" {
		options.prefix, err = prompt.Ask("Prefix", "key")
		if err != nil {
			return err
		}
	}
	options.copy, err = prompt.Confirm("Copy to clipboard?", false)
	if err != nil {
		return err
	}
	print, err := prompt.Confirm("Print to terminal?", true)
	if err != nil {
		return err
	}
	options.noPrint = !print
	return runSecret(cmd, kind, options)
}

func newSecretSubcommand(kind string, short string, defaultLength int, defaultFormat string) *cobra.Command {
	options := secretOptions{length: defaultLength, envKey: defaultSecretKey(kind), format: defaultFormat, symbols: true}
	command := &cobra.Command{
		Use:   kind,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSecret(cmd, kind, options)
		},
	}
	command.Flags().IntVar(&options.length, "length", defaultLength, "random byte or character length")
	command.Flags().BoolVar(&options.copy, "copy", false, "copy generated secret to clipboard")
	command.Flags().BoolVar(&options.noPrint, "no-print", false, "do not print generated secret")
	if kind == "jwt" {
		command.Flags().StringVar(&options.format, "format", defaultFormat, "hex, base64, or env")
		command.Flags().StringVar(&options.envKey, "key", "JWT_SECRET", ".env key name")
	}
	if kind == "env" {
		command.Flags().StringVar(&options.format, "format", defaultFormat, "hex or base64")
		command.Flags().StringVar(&options.envKey, "key", "SECRET", ".env key name")
	}
	if kind == "password" {
		command.Flags().BoolVar(&options.symbols, "symbols", true, "include symbol characters")
	}
	if kind == "api-key" {
		command.Flags().StringVar(&options.prefix, "prefix", "key", "API key prefix")
	}
	return command
}

func runSecret(cmd *cobra.Command, kind string, options secretOptions) error {
	var (
		value string
		err   error
	)
	switch kind {
	case "password":
		value, err = kitsecret.PasswordWithOptions(options.length, options.symbols)
	case "token":
		value, err = kitsecret.Token(options.length)
	case "api-key":
		value, err = kitsecret.APIKey(options.prefix, options.length)
	case "jwt":
		value, err = kitsecret.JWTWithFormat(options.length, options.format, options.envKey)
	case "hex":
		value, err = kitsecret.Hex(options.length)
	case "base64":
		value, err = kitsecret.Base64(options.length)
	case "env":
		value, err = secretValueByFormat(options.length, options.format)
		if err == nil {
			value, err = kitsecret.EnvLine(options.envKey, value)
		}
	default:
		err = fmt.Errorf("unsupported secret type: %s", kind)
	}
	if err != nil {
		return err
	}

	command := secretCommandPreview(kind, options)
	if options.copy {
		command += " --copy"
	}
	if options.noPrint {
		command += " --no-print"
	}

	summary := secretSummary(kind, options)
	if options.copy {
		if err := kitsecret.CopyToClipboard(context.Background(), value); err != nil {
			summary += "\nClipboard copy failed: " + err.Error()
		} else {
			summary += "\nCopied to clipboard."
		}
	}
	result := value
	if options.noPrint {
		result = "(hidden)"
	}

	return writer(cmd).Write(output.Result{
		Title:   "Secret " + secretTitle(kind),
		Command: []string{command},
		Summary: summary,
		Result:  result,
	})
}

func secretValueByFormat(length int, format string) (string, error) {
	switch format {
	case "hex":
		return kitsecret.Hex(length)
	case "base64":
		return kitsecret.Base64(length)
	default:
		return "", fmt.Errorf("unsupported secret format: %s", format)
	}
}

func secretCommandPreview(kind string, options secretOptions) string {
	parts := []string{"kit internal secret", kind, "--length", fmt.Sprint(options.length)}
	switch kind {
	case "jwt":
		parts = append(parts, "--format", options.format)
		if options.format == "env" {
			parts = append(parts, "--key", kitsecret.NormalizeEnvKey(options.envKey))
		}
	case "env":
		parts = append(parts, "--format", options.format, "--key", kitsecret.NormalizeEnvKey(options.envKey))
	case "password":
		if !options.symbols {
			parts = append(parts, "--symbols=false")
		}
	case "api-key":
		if options.prefix != "" {
			parts = append(parts, "--prefix", options.prefix)
		}
	}
	return strings.Join(parts, " ")
}

func secretSummary(kind string, options secretOptions) string {
	lines := []string{"Generated with crypto/rand.", "Secret is not stored by kit.", "Printing secrets can leave terminal scrollback or shell logs behind."}
	switch kind {
	case "password", "token":
		lines = append(lines, fmt.Sprintf("Length: %d characters", options.length))
	case "api-key":
		lines = append(lines, fmt.Sprintf("Token length: %d characters", options.length), "Prefix: "+options.prefix)
	case "hex":
		lines = append(lines, fmt.Sprintf("Entropy: %d bytes", options.length), fmt.Sprintf("Output length: %d hex characters", options.length*2))
	case "base64":
		lines = append(lines, fmt.Sprintf("Entropy: %d bytes", options.length), "Encoding: base64url without padding")
	case "jwt":
		lines = append(lines, fmt.Sprintf("Entropy: %d bytes", options.length), "Format: "+options.format)
	case "env":
		lines = append(lines, fmt.Sprintf("Entropy: %d bytes", options.length), "Key: "+kitsecret.NormalizeEnvKey(options.envKey))
	}
	return strings.Join(lines, "\n")
}

func defaultSecretFormat(kind string) string {
	switch kind {
	case "jwt", "hex":
		return "hex"
	case "base64", "env":
		return "base64"
	default:
		return "text"
	}
}

func defaultSecretKey(kind string) string {
	if kind == "jwt" {
		return "JWT_SECRET"
	}
	return "SECRET"
}

func secretTitle(kind string) string {
	switch kind {
	case "jwt":
		return "JWT"
	case "uuid":
		return "UUID"
	case "api-key":
		return "API Key"
	default:
		return titleAction(kind)
	}
}

func uuidCommandPreview(options secretOptions) string {
	command := "kit internal secret uuid"
	if options.copy {
		command += " --copy"
	}
	if options.noPrint {
		command += " --no-print"
	}
	return command
}
