package command

import (
	"fmt"
	"github.com/dixonwille/wmenu/v5"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(configLangCmd)
}

var configLangCmd = &cobra.Command{
	Use:   "config",
	Short: "Application configuration",
	Long:  `Menu for setting up the application.`,
	Run: func(cmd *cobra.Command, args []string) {
		ConfigMenu()
	},
}

func ConfigMenu() {
	wmenu.Clear()
	menu := wmenu.NewMenu("Choose a configuration:")
	menu.LoopOnInvalid()

	menu.Option("Language settings", nil, false, func(option wmenu.Opt) error {
		SetLang()
		return nil
	})

	menu.Run()
}

func SetLang() {
	wmenu.Clear()
	menu := wmenu.NewMenu("Select application language:")
	menu.LoopOnInvalid()

	menu.Action(func(opts []wmenu.Opt) error { fmt.Println(opts[0].Value); return nil })

	menu.Option("English", "en", true, func(opt wmenu.Opt) error {
		SaveLangConfig(opt.Value)
		return nil
	})
	menu.Option("Russian", "ru", false, func(opt wmenu.Opt) error {
		SaveLangConfig(opt.Value)
		return nil
	})

	menu.Run()
}

func SaveLangConfig(lang interface{}) {
	viper.Set("lang", lang)
	viper.WriteConfig()

	switch lang {
	case "ru":
		fmt.Println("Выбран русский язык")
	case "en":
		fmt.Println("English language selected")
	}
}
