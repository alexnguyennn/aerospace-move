package main

import (
	"fmt"
	"os"
	"strings"

	path "path/filepath"

	dhall "github.com/philandstuff/dhall-golang/v6"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/bitfield/script"
)

func main() {
	var cfgFile string

	var rootCmd = &cobra.Command{Use: "yabai_move"}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.myapp.yaml)")

	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))

	rootCmd.AddCommand(cmdHello)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

type WindowRule struct {
	Title      string `dhall:"title"`
	TitleRegex string `dhall:"titleRegex"`
	AppRegex   string `dhall:"appRegex"`
	App        string `dhall:"app"`
}

type Space struct {
	Index uint8 `dhall:"index"`
	Name  string `dhall:"name"`
	Rules []WindowRule `dhall:"rules"`
}

type Spaces struct {
	Spaces []Space `dhall:"spaces"`
}

type Window struct {
	ID uint64 `json:"id"`
	App string `json:"app"`
	Title string `json:"title"`
}


var cmdHello = &cobra.Command{
	Use:   "hello",
	Short: "Say hello",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Hello, World!")
		configPath := os.Getenv("YM_FILE")
		// TODO; make this use a dhall file validated and generated from yaml
		// maybe use exec and reader unmarshall method to try
		if len(configPath) == 0 {
			xdgHome := os.Getenv("XDG_CONFIG_HOME")
			if len(xdgHome) == 0 {
				xdgHome = fmt.Sprintf("%s/.config", os.Getenv("HOME"))
			}
			configPath = path.Join(xdgHome, "yabai", "yabai-move", "work-desk.dhall")
		}

		var spacesConfig Spaces
		err := dhall.UnmarshalFile(configPath, &spacesConfig)
		if err != nil {
			panic(err)
		}

		windowsString, err := script.Exec("yabai -m query --windows").String()
		if err != nil {
			panic(err)
		}

		moveMap := map[uint8][]string{}

		for _, space := range spacesConfig.Spaces {
			moveMap[space.Index] = []string{}

			for _, rule := range space.Rules {
				//.[] | select((."title" | test("\\[work\\]")) and (."app" | test("firefox"; "i"))) | { t: .title, t: .app }
				// map(select((."title" | test("\\[work\\]")) and (."app" | test("firefox"; "i")))) | if [.] | length >= 1 then { id: .id, app: .app, title: .title }  else "" end
				//map(select((."title" | test("\\[work\\]")) and (."app" | test("firefox"; "i"))))
				//map(select((."title" | test("\\[work\\]")) and (."app" | test("firefox"; "i")))) | if (. | length == 1) then .[] | {id : .id, app: .app, title: .title } else "" end
					//`.[] | select((."title" | test("%s")) and (."app" | test("%s"; "i"))) | { id: .id, app: .app, title: .title }`,
				// TODO; leave alone if space index is already correct
				filterString := fmt.Sprintf(
					`map(select((."title" | test("%s"; "i")) and (."app" | test("%s"; "i")))) | if (. | length == 1) then .[] | {id : .id, app: .app, title: .title } else null end`,
					rule.TitleRegex, rule.AppRegex)
				// windowMatches, err := script.Echo(windowsString).JQ(filterString).String()
				window, err := script.Echo(windowsString).JQ(filterString).String()
				if err != nil {
					panic(err)
				}

				if strings.Contains(window, "null") || len(window) == 0 {
					continue
				}

				app, err := script.Echo(window).JQ(".app").String()
				if err != nil {
					panic(fmt.Errorf("rule: %+v, win: %d, err:%s", rule, len(window), err.Error()))
				}

				title, err := script.Echo(window).JQ(".title").String()
				if err != nil {
					panic(err)
				}

				id, err := script.Echo(window).JQ(".id").String()
				if err != nil {
					panic(err)
				}

				fmt.Printf("[DEBUG] Space: %d rules - titleRegex: %s | appRegex: %s -> matched title: %s, app: %s, id: %s",
					space.Index, rule.TitleRegex, rule.AppRegex, app, title, id,
				)

				moveMap[space.Index] = append(moveMap[space.Index], id)
			}

			for spaceIndex, ids := range moveMap {
				for _, id := range ids {
					script.Exec(fmt.Sprintf(
						"yabai -m window %s --space %d", id, spaceIndex,
					)).Stdout()
				}
			}

		}

	},
}

func init() {
	cobra.OnInitialize(initConfig)
}

func initConfig() {
	if viper.GetString("config") != "" {
		viper.SetConfigFile(viper.GetString("config"))
		viper.ReadInConfig()
	}
}
