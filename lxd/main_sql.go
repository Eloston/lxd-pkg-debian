package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/lxc/lxd/client"
)

type cmdSql struct {
	global *cmdGlobal
}

func (c *cmdSql) Command() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Use = "sql <query>"
	cmd.Short = "Execute a SQL query against the LXD database"
	cmd.Long = `Description:
  Execute a SQL query against the LXD database

  This internal command is mostly useful for debugging and disaster
  recovery. The LXD team will occasionally provide hotfixes to users as a
  set of database queries to fix some data inconsistency.

  This command targets the global LXD database and works in both local
  and cluster mode.
`
	cmd.RunE = c.Run
	cmd.Hidden = true

	return cmd
}

func (c *cmdSql) Run(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		cmd.Help()

		if len(args) == 0 {
			return nil
		}

		return fmt.Errorf("Missing required arguments")
	}

	query := args[0]

	// Connect to LXD
	d, err := lxd.ConnectLXDUnix("", nil)
	if err != nil {
		return err
	}

	data := internalSQLPost{
		Query: query,
	}
	response, _, err := d.RawQuery("POST", "/internal/sql", data, "")
	if err != nil {
		return err
	}

	result := internalSQLResult{}
	err = json.Unmarshal(response.Metadata, &result)
	if err != nil {
		return err
	}
	if strings.HasPrefix(strings.ToUpper(query), "SELECT") {
		// Print results in tabular format
		widths := make([]int, len(result.Columns))
		for i, column := range result.Columns {
			widths[i] = len(column)
		}
		for _, row := range result.Rows {
			for i, v := range row {
				width := 10
				switch v := v.(type) {
				case string:
					width = len(v)
				case int:
					width = 6
				case int64:
					width = 6
				case time.Time:
					width = 12
				}
				if width > widths[i] {
					widths[i] = width
				}
			}
		}
		format := "|"
		separator := "+"
		columns := make([]interface{}, len(result.Columns))
		for i, column := range result.Columns {
			format += " %-" + strconv.Itoa(widths[i]) + "v |"
			columns[i] = column
			separator += strings.Repeat("-", widths[i]+2) + "+"
		}
		format += "\n"
		separator += "\n"
		fmt.Printf(separator)
		fmt.Printf(fmt.Sprintf(format, columns...))
		fmt.Printf(separator)
		for _, row := range result.Rows {
			fmt.Printf(format, row...)
		}
		fmt.Printf(separator)
	} else {
		fmt.Printf("Rows affected: %d\n", result.RowsAffected)
	}
	return nil
}
