package cmd

import (
	"fmt"
	"regexp"
	"sort"

	"github.com/chiselstrike/iku-turso-cli/internal"
	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
	"unicode/utf8"
)

func init() {
	dbCmd.AddCommand(regionsCmd)
	addLatencyFlag(regionsCmd)
}

var regionsCmd = &cobra.Command{
	Use:               "locations",
	Short:             "List available database locations.",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		client, err := createTursoClientFromAccessToken(true)
		if err != nil {
			return err
		}
		locations, err := locations(client)
		if err != nil {
			return err
		}

		closest, err := closestLocation(client)
		if err != nil {
			return err
		}

		columns := make([]interface{}, 0)

		lats := make(map[string]int)
		var ids []string
		if latencyFlag {
			lats, err = latencies(client)
			if err != nil {
				return err
			}
			ids = maps.Keys(lats)
			sort.Slice(ids, func(i, j int) bool {
				return lats[ids[i]] < lats[ids[j]]
			})
			columns = append(columns, "ID")
			columns = append(columns, "LOCATION")
			columns = append(columns, "LATENCY↓")
		} else {
			ids = maps.Keys(locations)
			sort.Strings(ids)
			columns = append(columns, "ID↓")
			columns = append(columns, "LOCATION")
		}

		regex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
		tbl := table.New(columns...).WithWidthFunc(func(s string) int {
			plainText := regex.ReplaceAllString(s, "")
			return utf8.RuneCountInString(plainText)
		})

		for _, location := range ids {
			description := locations[location]
			lat, ok := lats[location]
			var latency string
			if ok {
				latency = fmt.Sprintf("%dms", lat)
			} else {
				latency = "???"
			}

			if location == closest {
				description = fmt.Sprintf("%s  [default]", description)
				if latencyFlag {
					tbl.AddRow(internal.Emph(location), internal.Emph(description), internal.Emph(latency))
				} else {
					tbl.AddRow(internal.Emph(location), internal.Emph(description))
				}
			} else {
				if latencyFlag {
					tbl.AddRow(location, description, latency)
				} else {
					tbl.AddRow(location, description)
				}
			}
		}
		tbl.Print()
		return nil
	},
}
