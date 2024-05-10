package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"go.fabry.dev/fbot/bazos"
)

func main() {
	var loglvl string

	rootCmd := &cobra.Command{
		Use:          "bazos",
		Short:        "A CLI app for bazos.sk",
		Long:         `A CLI app for interacting with the popular advertising website bazos.sk`,
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
			lvl := logrus.InfoLevel
			if loglvl != "" {
				lvl, err = logrus.ParseLevel(loglvl)
				if err != nil {
					return err
				}
			}
			logrus.SetLevel(lvl)
			if logrus.IsLevelEnabled(logrus.DebugLevel) {
				logrus.SetReportCaller(true)
			}
			return nil
		},
	}

	rootCmd.PersistentFlags().StringVarP(&loglvl, "loglvl", "L", "", "Set logging level (trace, debug, info, warn, error)")

	// Search command
	searchCmd := &cobra.Command{
		Use:          "search [query]",
		Short:        "Searches bazos.sk for ads",
		Long:         `Searches bazos.sk for ads matching the given query.`,
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")
			ads, err := bazos.Search(query)
			if err != nil {
				return err
			}

			for i, ad := range ads {
				fmt.Printf("- #%d - %v - %v€ (ID: %v) - %v\n", i+1, ad.Title, ad.Price, ad.ID, ad.Link)
			}
			return nil
		},
	}
	searchCmd.Flags().StringP("category", "c", "", "Category to search within (format: 'category/section')")
	searchCmd.Flags().StringP("location", "l", "", "Location to search in")
	searchCmd.Flags().IntP("vicinity", "v", 0, "Radius in km for vicinity search")
	searchCmd.Flags().StringP("price", "p", "", "Price range to search within (format: 'min-max')")

	rootCmd.AddCommand(searchCmd)

	// GetAdById ad listing command
	getCmd := &cobra.Command{
		Use:          "get [id]",
		Short:        "Retrieves specific ad listing",
		Long:         `Retrieves specific ad listing with the given ID.`,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			ad, err := bazos.GetAdById(id)
			if err != nil {
				return err
			}
			fmt.Printf("%+v\n", toYaml(ad))
			return nil
		},
	}

	rootCmd.AddCommand(getCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func toYaml(v any) string {
	var buf strings.Builder
	enc := yaml.NewEncoder(&buf)
	if err := enc.Encode(v); err != nil {
		return err.Error()
	}
	return buf.String()
}
