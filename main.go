package main

import (
	"aerospace_move/pkg/schema"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	path "path/filepath"
	"regexp"
	"time"

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

	rootCmd.PersistentFlags().Bool(
		"pause",
		false,
		"toggle to pause on complete",
	)

	err := viper.BindPFlag("pkl", rootCmd.PersistentFlags().Lookup("pkl"))
	if err != nil {
		panic(err)
	}

	err = viper.BindPFlag("dry-run", rootCmd.PersistentFlags().Lookup("dry-run"))
	if err != nil {
		panic(err)
	}
	err = viper.BindPFlag("pause", rootCmd.PersistentFlags().Lookup("pause"))
	if err != nil {
		panic(err)
	}

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type AerospaceWindow struct {
	AppBundleID string `json:"app-bundle-id"`
	AppPID      int    `json:"app-pid"`
	WindowTitle string `json:"window-title"`
	AppName     string `json:"app-name"`
	WindowID    uint64 `json:"window-id"`
	Workspace   string `json:"workspace"`
}

var cmd = &cobra.Command{
	Use:   "aerospace-move",
	Short: "runs aerospace move based on given config as pkl file",
	RunE: func(cmd *cobra.Command, args []string) error {
		amCfg, err := parsePklConfig()
		if err != nil {
			return err
		}

		err = attemptRestarts(amCfg.RestartConfig)
		if err != nil {
			return err
		}

		windows, err := getWindows()
		if err != nil {
			return err
		}

		// first
		alreadyInWorkspaceRegex, err := regexp.Compile(`Window\s'.*'\salready\sbelongs\sto\sworkspace\s'.*'`)
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
					windows, func(w AerospaceWindow, _ int) bool {
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

						shouldMove := w.Workspace != space.Name

						return acmp && tcmp && shouldMove
					},
				)
				for _, w := range spaceMatchedWin {
					_, err = fmt.Fprintln(
						cmd.OutOrStdout(),
						fmt.Sprintf("DEBUG: matched rule, sending w: %+v to workspace: %s", w, space.Name),
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
							`aerospace move-node-to-workspace --window-id '%d' '%s'`,
							w.WindowID,
							space.Name,
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
		if viper.GetBool("pause") {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "DEBUG: pausing on complete...")
			select {}
		}
		return err
	},
}

type asyncResult struct {
	success bool
	err     error
}

func attemptRestarts(restartConfig []*schema.WindowRule) error {
	_, _ = fmt.Fprintln(os.Stderr, "DEBUG: attempting to restart on config:", restartConfig)
	windows, err := getWindows()
	if err != nil {
		return err
	}
	mapErrs := make([]error, len(restartConfig)*2)
	_, _ = fmt.Fprintln(os.Stderr, "DEBUG: attempting to restart on config:", restartConfig)
	matchedWins := lo.FlatMap(
		restartConfig, func(rule *schema.WindowRule, _ int) []AerospaceWindow {
			var tRegex *regexp.Regexp
			var aRegex *regexp.Regexp
			if !lo.IsEmpty(rule.TitleRegex) {
				tRegex, err = regexp.Compile(rule.TitleRegex)
				if err != nil {
				}
				mapErrs = append(mapErrs, err)
			}
			if !lo.IsEmpty(rule.AppRegex) {
				aRegex, err = regexp.Compile(rule.AppRegex)
				if err != nil {
					mapErrs = append(mapErrs, err)
				}
			}
			return lo.Filter(
				windows, func(w AerospaceWindow, _ int) bool {
					//_, _ = fmt.Fprintln(os.Stderr, "DEBUG: matching rule against window:", rule, w)
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

					return acmp && tcmp
				},
			)
		},
	)
	//_, _ = fmt.Fprintln(os.Stderr, "DEBUG: matching windows", matchedWins)
	if len(mapErrs) > 0 {
		//_, _ = fmt.Fprintln(os.Stderr, "DEBUG: got map errs while matching for restart", mapErrs)
		err = errors.Join(mapErrs...)
	}
	if err != nil {
		return err
	}
	return restartAppMatches(matchedWins)
}

func restartAppMatches(matches []AerospaceWindow) error {
	isDryRun := viper.GetBool("dry-run")
	appMatches := lo.Uniq(
		lo.Map(
			matches, func(w AerospaceWindow, _ int) lo.Tuple2[string, int] {
				return lo.Tuple2[string, int]{A: w.AppBundleID, B: w.AppPID}
			},
		),
	)
	_, _ = fmt.Fprintln(os.Stderr, "DEBUG: matching apps", appMatches)

	rCh := make(chan asyncResult)
	for _, m := range appMatches {
		_, _ = fmt.Fprintln(os.Stderr, "DEBUG: restarting app:", m.A, m.B)
		if isDryRun {
			// skip running
			continue
		}
		go restartApp(m.A, m.B, rCh)
	}

	if isDryRun {
		// skip waiting for result
		return nil
	}

	for i := 0; i < len(appMatches); i++ {
		select {
		case rRes := <-rCh:
			fmt.Println("DEBUG: received a restart app attempt completed event", rRes)
			if !rRes.success && (errors.Is(rRes.err, ErrBlockedQuit) || errors.Is(rRes.err, ErrBlockedStart)) {
				// block and wait for user to quit app
				_, _ = fmt.Fprintln(os.Stderr, rRes.err.Error())
				select {}
			}
			if !rRes.success && rRes.err != nil {
				return rRes.err
			}
			if !rRes.success && rRes.err == nil {
				return errors.New("unexpected error: restartedApp")
			}
			// success, wait for next one
			continue
		}
	}
	return nil
}

func restartApp(appBundleID string, appPID int, rCh chan asyncResult) {
	sCh := make(chan asyncResult)
	qCh := make(chan asyncResult)
	go quitApp(appBundleID, appPID, qCh)
	qRes := <-qCh
	if !qRes.success {
		rCh <- qRes
	}
	go startApp(appBundleID, sCh)
	sRes := <-sCh
	rCh <- sRes
}

var ErrBlockedStart error = errors.New("app blocked from starting")
var ErrBlockedQuit error = errors.New("app blocked from quitting")

const waitIncrement = 250 * time.Millisecond
const waitAttempts = 100

func startApp(appBundleID string, ch chan asyncResult) {
	outBuf := bytes.Buffer{}
	_, err := script.Exec(
		fmt.Sprintf(
			`open -b '%s'`,
			appBundleID,
		),
	).WithStderr(&outBuf).Stdout()
	if err != nil {
		ch <- asyncResult{success: false, err: fmt.Errorf(
			`error opening app %s buf %s : %w`,
			appBundleID,
			outBuf.String(),
			err,
		)}
	}

	for i, _ := range lo.Range(waitAttempts) {
		outBuf = bytes.Buffer{}
		_, err := script.Exec(
			fmt.Sprintf(
				`aerospace list-windows --app-bundle-id '%s' --monitor all`,
				appBundleID,
			),
		).WithStdout(&outBuf).Stdout()
		if err != nil {
			ch <- asyncResult{success: false, err: fmt.Errorf(
				`error checking app %s buf %s : %w`,
				appBundleID,
				outBuf.String(),
				err,
			)}
		}

		if outBuf.String() != "" {
			// app has started; bundle shows up in list
			ch <- asyncResult{success: true, err: nil}
		}

		time.Sleep(waitIncrement)
		_, _ = fmt.Fprintln(os.Stderr, "DEBUG: waiting for app to start, attempt:", i, appBundleID, outBuf.String())
	}
	ch <- asyncResult{success: false, err: fmt.Errorf(
		"%s did not start after %d attempts: %w",
		appBundleID,
		waitAttempts,
		ErrBlockedStart,
	)}
}

func quitApp(appBundleID string, appPID int, ch chan asyncResult) {
	outBuf := bytes.Buffer{}
	_, err := script.Exec(
		fmt.Sprintf(
			`osascript -e '
try
	tell application id "%s" to quit
on error errMsg number errNbr
	do shell script "echo " & quoted form of errMsg & " & echo Error number " & errNbr
end try
'`,
			appBundleID,
		),
	).WithStderr(&outBuf).Stdout()
	if err != nil {
		ch <- asyncResult{success: false, err: fmt.Errorf(
			`error quitting app %s buf %s : %w`,
			appBundleID,
			outBuf.String(),
			err,
		)}
	}

	for i, _ := range lo.Range(waitAttempts) {
		outBuf = bytes.Buffer{}
		_, err := script.Exec(
			fmt.Sprintf(
				`aerospace list-windows --pid '%d' --monitor all`,
				appPID,
			),
		).WithStdout(&outBuf).Stdout()
		if err != nil {
			ch <- asyncResult{success: false, err: err}
			ch <- asyncResult{success: false, err: fmt.Errorf(
				`error checking app pid %d %s buf:' %s : %w`,
				appPID,
				appBundleID,
				outBuf.String(),
				err,
			)}
		}

		if outBuf.String() == "" {
			// app has quit; pid doesn't exist anymore
			ch <- asyncResult{success: true, err: nil}
		}

		time.Sleep(waitIncrement)
		//time.Sleep(waitAttempts * waitIncrement) // wait longer and longer for app to quit
		_, _ = fmt.Fprintln(
			os.Stderr,
			"DEBUG: waiting for app to quit, attempt:",
			i,
			appPID,
			appBundleID,
			outBuf.String(),
		)
	}
	ch <- asyncResult{success: false, err: fmt.Errorf(
		"%d: %s did not quit after %d attempts: %w",
		appPID,
		appBundleID,
		waitAttempts,
		ErrBlockedQuit,
	)}
}

func getWindows() ([]AerospaceWindow, error) {
	windowStr, err := script.Exec(
		fmt.Sprintf(
			`aerospace list-windows --all --json --format "%%{app-pid} %%{app-name} %%{app-bundle-id} %%{window-title} %%{window-id} %%{workspace}"`,
		),
	).String()
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "WARN: error getting windows for workspace: ", err)
		return nil, err
	}

	var workspaceWindows []AerospaceWindow
	err = json.Unmarshal([]byte(windowStr), &workspaceWindows)
	if err != nil {
		_, _ = fmt.Fprintln(
			os.Stderr,
			"WARN: error getting aerospace windows:",
			err,
			windowStr,
		)
		return nil, err
	}

	_, err = fmt.Fprintln(os.Stderr, "DEBUG: processed windows:", workspaceWindows)
	return workspaceWindows, err
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
