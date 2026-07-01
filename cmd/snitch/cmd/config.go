package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/fristovic/snitch/internal/config"
	"github.com/fristovic/snitch/internal/ipc"
	"github.com/fristovic/snitch/internal/platform"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or update configuration",
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a config value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		paths, _ := platform.Resolve()
		cfg, err := config.Load(paths.ConfigPath)
		if err != nil {
			return err
		}
		val, err := cfg.Get(args[0])
		if err != nil {
			return err
		}
		fmt.Println(val)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a config value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := ipc.Connect(resolveSocket())
		if err != nil {
			paths, _ := platform.Resolve()
			cfg, err := config.Load(paths.ConfigPath)
			if err != nil {
				return err
			}
			if err := cfg.Set(args[0], args[1]); err != nil {
				return err
			}
			return cfg.Save(paths.ConfigPath)
		}
		defer client.Close()
		_, err = client.Call("set_config", map[string]string{"key": args[0], "value": args[1]})
		return err
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show full configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := ipc.Connect(resolveSocket())
		if err != nil {
			paths, _ := platform.Resolve()
			cfg, err := config.Load(paths.ConfigPath)
			if err != nil {
				return err
			}
			data, _ := json.MarshalIndent(cfg, "", "  ")
			fmt.Println(string(data))
			return nil
		}
		defer client.Close()
		data, err := client.Call("get_config", nil)
		if err != nil {
			return err
		}
		var pretty map[string]any
		_ = json.Unmarshal(data, &pretty)
		out, _ := json.MarshalIndent(pretty, "", "  ")
		fmt.Println(string(out))
		return nil
	},
}

func init() {
	configCmd.AddCommand(configGetCmd, configSetCmd, configShowCmd)
}
