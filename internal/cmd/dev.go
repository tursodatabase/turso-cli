package cmd

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"

	"database/sql"
	"github.com/chiselstrike/iku-turso-cli/internal"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/cobra"
	"io/ioutil"
	"os"
)

func init() {
	rootCmd.AddCommand(devCmd)
	addDevPortFlag(devCmd)
	addDevFileFlag(devCmd)
	addVerboseFlag(devCmd)
}

type requestBody struct {
	Statements []string `json:"statements"`
}

type successResponse struct {
	Results []struct {
		Columns []string        `json:"columns"`
		Rows    [][]interface{} `json:"rows"`
	} `json:"results"`
}

type errorResponse struct {
	Error string `json:"error"`
}

var devCmd = &cobra.Command{
	Use:               "dev",
	Short:             "starts a development server for Turso",
	Args:              cobra.NoArgs,
	ValidArgsFunction: noFilesArg,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		port := fmt.Sprintf(":%d", devPort)
		fmt.Printf("Notice that for environments where a filesystem is available, you can just use %s as a URL and this server is not needed.\n", internal.Emph("file:somefile.db"))
		fmt.Printf("For others environments, you can now connect to %s%s...\n", internal.Emph("http://localhost"), internal.Emph(port))

		gin.SetMode(gin.ReleaseMode)
		r := gin.New()
		if !verboseFlag {
			r.Use(gin.LoggerWithWriter(ioutil.Discard))
		}

		if devFile == "" {
			devFile = ":memory:"
		}
		db, err := sql.Open("sqlite3", devFile)
		if err != nil {
			fmt.Println("Failed to connect to SQLite database:", err)
			os.Exit(1)
		}
		defer db.Close()

		r.POST("/", func(c *gin.Context) {
			var reqBody requestBody

			err := c.BindJSON(&reqBody)
			if err != nil {
				c.JSON(http.StatusBadRequest, errorResponse{err.Error()})
				return
			}

			statements := reqBody.Statements

			var results []struct {
				Columns []string        `json:"columns"`
				Rows    [][]interface{} `json:"rows"`
			}

			for _, statement := range statements {
				if verboseFlag {
					fmt.Printf("Executing %s\n", internal.Emph(statement))
				}
				rows, err := db.Query(statement)

				if err != nil {
					c.JSON(http.StatusBadRequest, errorResponse{err.Error()})
					return
				}
				defer rows.Close()

			response := successResponse{
				Results: results,
			}
			c.JSON(http.StatusOK, response)
		})

		r.Run(port)
		return nil
	},
}
