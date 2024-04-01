package main

import (
	"aerospace_move/pkg/schema"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	path "path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/bitfield/script"
	"github.com/samber/lo"
)

// pkl-gen-go  pkl/schema.pkl --base-path github.com/alexnguyennn/aerospace-move

func main() {
	var rootCmd = cmd

	rootCmd.PersistentFlags().String(
		"pkl",
		getDefaultPklPath(),
		"pkl file",
	)

	rootCmd.PersistentFlags().Bool(
		"dry-run",
		false,
		"toggle to only show debug logs",
	)

	err := viper.BindPFlag("pkl", rootCmd.PersistentFlags().Lookup("pkl"))
	if err != nil {
		panic(err)
	}

	err = viper.BindPFlag("dry-run", rootCmd.PersistentFlags().Lookup("dry-run"))
	if err != nil {
		panic(err)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type Window struct {
	PID   uint64
	App   string
	Title string
}

var cmd = &cobra.Command{
	Use:   "aerospace-move",
	Short: "runs aerospace move based on given config as pkl file",
	RunE: func(cmd *cobra.Command, args []string) error {
		amCfg, err := parsePklConfig()
		if err != nil {
			return err
		}

		windowStr, err := script.Exec(
			fmt.Sprintf(
				"aerospace list-windows --all",
			),
		).Slice()
		if err != nil {
			return err
		}

		windows, err := processWindows(windowStr)
		if err != nil {
			return err
		}
		// first
		alreadyInWorkspaceRegex, err := regexp.Compile(`Window\s'.*'\salready\sbelongs\sto\sworkspace\s'.*'`)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(cmd.ErrOrStderr(), "processed windows:", windows)
		if err != nil {
			return err
		}

		for _, space := range amCfg.MoveConfig {
			for _, rule := range space.Rules {
				var tRegex *regexp.Regexp
				var aRegex *regexp.Regexp
				if !lo.IsEmpty(rule.TitleRegex) {
					tRegex, err = regexp.Compile(rule.TitleRegex)
					if err != nil {
						return err
					}
				}
				if !lo.IsEmpty(rule.AppRegex) {
					aRegex, err = regexp.Compile(rule.AppRegex)
					if err != nil {
						return err
					}
				}
				spaceMatchedWin := lo.Filter(
					windows, func(w Window, _ int) bool {
						if lo.IsEmpty(w.Title) && lo.IsEmpty(w.App) {
							return false
						}

						acmp := lo.TernaryF(
							lo.IsNil(aRegex),
							func() bool { return w.App == rule.App },
							func() bool { return aRegex.MatchString(w.App) },
						)
						tcmp := lo.TernaryF(
							lo.IsNil(tRegex),
							func() bool { return w.Title == rule.Title },
							func() bool { return tRegex.MatchString(w.Title) },
						)

						return acmp && tcmp
					},
				)
				for _, w := range spaceMatchedWin {
					_, err = fmt.Fprintln(
						cmd.OutOrStdout(),
						fmt.Sprintf("DEBUG: matched rule, sending w: %+v to workspace: %d", w, space.Index),
					)
					if err != nil {
						return err
					}

					if viper.GetBool("dry-run") {
						// skip running
						continue
					}

					outBuf := bytes.Buffer{}
					_, err = script.Exec(
						fmt.Sprintf(
							`sh -c 'hs -c "focusWindowByPid(%d)" && aerospace move-node-to-workspace %d'`,
							w.PID, space.Index,
						),
					).WithStderr(&outBuf).Stdout()
					_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "DEBUG stdErr: got output:", outBuf.String())
					if err != nil && !alreadyInWorkspaceRegex.MatchString(outBuf.String()) {
						return err
					}
				}
			}
		}

		_, err = script.Exec(`hs -c 'showAlert("aerospace-move complete")'`).Stdout()
		return err
	},
}

func processWindows(lines []string) ([]Window, error) {
	var windows []Window
	for _, line := range lines {
		fields := strings.Split(line, "|")
		if len(fields) != 3 {
			_, err := fmt.Fprintln(
				os.Stderr,
				"WARN received line that splits into more fields than expected for a window: ",
				line,
			)
			if err != nil {
				return nil, err
			}
		}

		pid, err := strconv.ParseUint(strings.TrimSpace(fields[0]), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("error parsing PID: %w", err)
		}
		app := strings.TrimSpace(fields[1])
		var title = ""
		if len(fields) >= 3 {
			title = strings.TrimSpace(
				lo.Ternary(len(fields) > 3, strings.Join(fields[2:], ""), fields[2]),
			)
		}
		windows = append(windows, Window{PID: pid, App: app, Title: title})
	}
	return windows, nil
}

func getDefaultPklPath() string {
	xdgHome := os.Getenv("XDG_CONFIG_HOME")
	if len(xdgHome) == 0 {
		xdgHome = fmt.Sprintf("%s/.config", os.Getenv("HOME"))
	}
	return path.Join(xdgHome, "aerospace", "aerospace-move", "work-desk.pkl")
}

func parsePklConfig() (*schema.AerospaceMove, error) {
	pklPath := viper.GetString("pkl")
	if pklPath != "" {
		return schema.LoadFromPath(context.Background(), pklPath)
	}
	return nil, errors.New("got an empty pkl path when parsing config")
}
