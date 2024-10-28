package main

import (
	"aerospace_move/pkg/schema"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
	path "path/filepath"
	"regexp"

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
	Workspace uint8
	AerospaceWindow
}

type AerospaceWindow struct {
	AppBundleID string `json:"app-bundle-id"`
	AppPID      int    `json:"app-pid"`
	WindowTitle string `json:"window-title"`
	AppName     string `json:"app-name"`
	WindowID    uint64 `json:"window-id"`
}

var cmd = &cobra.Command{
	Use:   "aerospace-move",
	Short: "runs aerospace move based on given config as pkl file",
	RunE: func(cmd *cobra.Command, args []string) error {
		amCfg, err := parsePklConfig()
		if err != nil {
			return err
		}

		/*		// TODO: add in once window struct is fixed
				windows, err := getWindows()
				if err != nil {
					return err
				}

				for _, rule := range amCfg.RestartConfig {
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
					matchedWins := lo.Filter(
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

					// TOOD: grab bundle id info
					matchedAppBundleIDs := lo.Uniq(
						lo.Map(
							matchedWins, func(w Window, _ int) string {
								return w.App
							},
						),
					)
					for _, a := range matchedAppBundleIDs {
						_, err = fmt.Fprintln(
							cmd.OutOrStdout(),
							fmt.Sprintf("DEBUG: matched rule, restarting app %s", a),
						)
						if err != nil {
							return err
						}

						if viper.GetBool("dry-run") {
							// skip running
							continue
						}
						// TODO: add quit and restart logic
					}
				}*/

		windows, err := getWindows()
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
						if lo.IsEmpty(w.WindowTitle) && lo.IsEmpty(w.AppName) {
							return false
						}

						acmp := lo.TernaryF(
							lo.IsNil(aRegex),
							func() bool { return w.AppName == rule.App },
							func() bool { return aRegex.MatchString(w.AppName) },
						)
						tcmp := lo.TernaryF(
							lo.IsNil(tRegex),
							func() bool { return w.WindowTitle == rule.Title },
							func() bool { return tRegex.MatchString(w.WindowTitle) },
						)

						shouldMove := w.Workspace != space.Index

						return acmp && tcmp && shouldMove
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

					// TODO: try goroutines for this and see if it goes any faster
					outBuf := bytes.Buffer{}
					_, err = script.Exec(
						fmt.Sprintf(
							`aerospace move-node-to-workspace --window-id '%d' '%d'`,
							w.WindowID,
							space.Index,
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

func getWindows() ([]Window, error) {
	nLines, err := script.Exec(`aerospace list-workspaces --all`).CountLines()
	if err != nil {
		return nil, err
	}

	errs := make([]error, nLines*2)
	windows := lo.FlatMap(
		lo.Range(nLines), func(_ int, idx int) []Window {
			windowStr, err := script.Exec(
				fmt.Sprintf(
					`aerospace list-windows --workspace "%d" --json --format "%%{app-pid} %%{app-name} %%{app-bundle-id} %%{window-title} %%{window-id}"`,
					idx,
				),
			).String()
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, "WARN: error getting windows for workspace: ", idx, err)
				errs[idx] = err
				return nil
			}

			var workspaceWindows []AerospaceWindow
			err = json.Unmarshal([]byte(windowStr), &workspaceWindows)
			if err != nil {
				_, _ = fmt.Fprintln(
					os.Stderr,
					"WARN: error getting aerospoace windows for workspace: ",
					idx,
					err,
					windowStr,
				)
				errs[idx] = err
				return nil
			}

			return lo.Map(
				workspaceWindows, func(w AerospaceWindow, _ int) Window {
					return Window{
						AerospaceWindow: w,
						Workspace:       uint8(idx),
					}
				},
			)
		},
	)

	windowGetErr := errors.Join(errs...)
	if windowGetErr != nil {
		return nil, windowGetErr
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
