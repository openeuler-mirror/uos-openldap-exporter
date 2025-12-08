package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"gitee.com/openeuler/uos-openldap-exporter/internal/collector"
	"gitee.com/openeuler/uos-openldap-exporter/internal/config"
	"gitee.com/openeuler/uos-openldap-exporter/internal/logger"
	"gitee.com/openeuler/uos-openldap-exporter/internal/server"
)

var (
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "uos-openldap-exporter",
	Short: "A Prometheus exporter for OpenLDAP",
	Long: `A Prometheus exporter for OpenLDAP that collects metrics from OpenLDAP server
and exposes them via HTTP for Prometheus to scrape.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load(cfgFile)
		log := logger.New(cfg.Log.Level, cfg.Log.Format)
		coll := collector.New(cfg, log)
		srv := server.New(cfg.Web.ListenAddress, cfg.Web.MetricsPath, coll, log)
		return srv.Run()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config.file", "", "Path to config file")

	// Web flags
	rootCmd.Flags().String("web.listen-address", ":9330", "Address to listen on")
	rootCmd.Flags().String("web.metrics-path", "/metrics", "Path under which to expose metrics")

	// LDAP flags
	rootCmd.Flags().String("ldap.server", "", "LDAP server URL (e.g., ldap://localhost:389)")
	rootCmd.Flags().String("ldap.bind-dn", "", "Bind DN for authentication")
	rootCmd.Flags().String("ldap.bind-password", "", "Bind password")

	// Bind viper flags
	if err := viper.BindPFlag("web.listen_address", rootCmd.Flags().Lookup("web.listen-address")); err != nil {
		panic(fmt.Errorf("failed to bind web.listen_address flag: %w", err))
	}
	if err := viper.BindPFlag("web.metrics_path", rootCmd.Flags().Lookup("web.metrics-path")); err != nil {
		panic(fmt.Errorf("failed to bind web.metrics_path flag: %w", err))
	}
	if err := viper.BindPFlag("ldap.server", rootCmd.Flags().Lookup("ldap.server")); err != nil {
		panic(fmt.Errorf("failed to bind ldap.server flag: %w", err))
	}
	if err := viper.BindPFlag("ldap.bind_dn", rootCmd.Flags().Lookup("ldap.bind-dn")); err != nil {
		panic(fmt.Errorf("failed to bind ldap.bind_dn flag: %w", err))
	}
	if err := viper.BindPFlag("ldap.bind_password", rootCmd.Flags().Lookup("ldap.bind-password")); err != nil {
		panic(fmt.Errorf("failed to bind ldap.bind_password flag: %w", err))
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Default config file name
		viper.SetConfigName("config")
	}

	viper.SetEnvPrefix("OPENLDAP_EXPORTER")
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func main() {
	Execute()
}
