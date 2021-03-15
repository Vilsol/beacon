package cmd

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/util/homedir"
	"os"
	"path/filepath"
	"time"
)

var rootCmd = &cobra.Command{
	Use:   "beacon",
	Short: "kubernetes beacon that restarts deployments when image updates",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		viper.SetConfigName("config")
		viper.AddConfigPath(".")
		viper.SetEnvPrefix("bot")
		viper.AutomaticEnv()

		_ = viper.ReadInConfig()

		zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

		level, err := zerolog.ParseLevel(viper.GetString("log"))
		if err != nil {
			return err
		}

		zerolog.SetGlobalLevel(level)

		if viper.GetBool("pretty") {
			log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
		}

		return nil
	},
}

func Execute() {
	// Execute transpile command as default
	cmd, _, err := rootCmd.Find(os.Args[1:])
	if (len(os.Args) <= 1 || os.Args[1] != "help") && (err != nil || cmd == rootCmd) {
		args := append([]string{"run"}, os.Args[1:]...)
		rootCmd.SetArgs(args)
	}

	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}

func init() {
	rootCmd.PersistentFlags().String("log", "info", "The log level to output")
	rootCmd.PersistentFlags().Bool("pretty", false, "Use pretty logging output (non-json)")

	rootCmd.PersistentFlags().String("label", "vilsol.beacon", "Label to filter deployments")
	rootCmd.PersistentFlags().Duration("interval", time.Minute*10, "Interval between update checks")
	rootCmd.PersistentFlags().StringSlice("namespaces", []string{""}, "List of namespaces to monitor (defaults to all)")

	defaultDir := ""
	if home := homedir.HomeDir(); home != "" {
		defaultDir = filepath.Join(home, ".kube", "config")
	}

	rootCmd.PersistentFlags().String("kubeconfig", defaultDir, "absolute path to the kubeconfig file")

	_ = viper.BindPFlag("log", rootCmd.PersistentFlags().Lookup("log"))
	_ = viper.BindPFlag("pretty", rootCmd.PersistentFlags().Lookup("pretty"))

	_ = viper.BindPFlag("label", rootCmd.PersistentFlags().Lookup("label"))
	_ = viper.BindPFlag("interval", rootCmd.PersistentFlags().Lookup("interval"))
	_ = viper.BindPFlag("namespaces", rootCmd.PersistentFlags().Lookup("namespaces"))

	_ = viper.BindPFlag("kubeconfig", rootCmd.PersistentFlags().Lookup("kubeconfig"))
}
