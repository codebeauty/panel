package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"github.com/codebeauty/panel/internal/config"
	"github.com/codebeauty/panel/internal/tui"
)

var jsonKeyRe = regexp.MustCompile(`^(\s*)"([^"]+)":`)

func colorizeJSON(line string) string {
	if m := jsonKeyRe.FindStringSubmatchIndex(line); m != nil {
		indent := line[:m[2*1+1]]
		key := line[m[2*2]:m[2*2+1]]
		rest := line[m[1]:]
		return indent + tui.StylePrimary.Render(`"`+key+`":`) + rest
	}
	return line
}

func newConfigCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "config",
		Short: "Show resolved configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			if tui.IsTTY() {
				fmt.Fprintf(os.Stderr, "%s %s\n\n",
					tui.StyleBold.Render("Config file:"),
					config.GlobalConfigPath())
			} else {
				fmt.Fprintf(os.Stderr, "Config file: %s\n\n", config.GlobalConfigPath())
			}

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			data, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling config: %w", err)
			}

			if tui.IsTTY() {
				// Simple syntax highlighting: keys in primary, strings in default, booleans/numbers in success
				lines := strings.Split(string(data), "\n")
				for _, line := range lines {
					fmt.Println(colorizeJSON(line))
				}
			} else {
				fmt.Println(string(data))
			}
			return nil
		},
	}
}
