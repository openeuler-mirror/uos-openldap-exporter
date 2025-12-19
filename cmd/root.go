package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"gitee.com/openeuler/uos-openldap-exporter/internal/collector"
	"gitee.com/openeuler/uos-openldap-exporter/internal/config"
	"gitee.com/openeuler/uos-openldap-exporter/internal/logger"
	"gitee.com/openeuler/uos-openldap-exporter/internal/server"
)

var (
	cfgFile string
	version string // 版本信息，可通过编译时注入
	commit  string // git提交信息，可通过编译时注入
	date    string // 构建日期，可通过编译时注入
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "uos-openldap-exporter",
	Short: "A Prometheus exporter for OpenLDAP",
	Long: `A Prometheus exporter for OpenLDAP that collects metrics from OpenLDAP server
and exposes them via HTTP for Prometheus to scrape.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}
		
		log := logger.New(cfg.Log.Level, cfg.Log.Format)
		coll := collector.New(cfg, log)
		srv := server.New(cfg.Web.ListenAddress, cfg.Web.MetricsPath, coll, log)
		return srv.Run()
	},
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version information",
	Long:  `Print the version information and exit`,
	Run: func(cmd *cobra.Command, args []string) {
		if version == "" {
			version = "dev"
		}
		if commit == "" {
			commit = "unknown"
		}
		if date == "" {
			date = "unknown"
		}
		fmt.Printf("uos-openldap-exporter Version: %s\nGit Commit: %s\nBuild Date: %s\n", version, commit, date)
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
	rootCmd.AddCommand(versionCmd)

	// Web flags
	rootCmd.Flags().String("web.listen-address", ":9330", "Address to listen on")
	rootCmd.Flags().String("web.metrics-path", "/metrics", "Path under which to expose metrics")

	// LDAP flags
	rootCmd.Flags().String("ldap.server", "", "LDAP server URL (e.g., ldap://localhost:389)")
	rootCmd.Flags().String("ldap.bind-dn", "", "Bind DN for authentication")
	rootCmd.Flags().String("ldap.bind-password", "", "Bind password")
	rootCmd.Flags().Duration("ldap.timeout", 10*time.Second, "LDAP connection timeout")
	rootCmd.Flags().Bool("ldap.start-tls", false, "Enable StartTLS")
	rootCmd.Flags().Bool("ldap.insecure-skip-verify", false, "Skip LDAP server certificate verification (NOT recommended for production)")

	// Log flags
	rootCmd.Flags().String("log.level", "info", "Log level (debug, info, warn, error)")
	rootCmd.Flags().String("log.format", "text", "Log format (text, json)")

	// Bind viper flags
	bindErrs := []error{}
	
	if err := viper.BindPFlag("web.listen_address", rootCmd.Flags().Lookup("web.listen-address")); err != nil {
		bindErrs = append(bindErrs, fmt.Errorf("failed to bind web.listen_address flag: %w", err))
	}
	if err := viper.BindPFlag("web.metrics_path", rootCmd.Flags().Lookup("web.metrics-path")); err != nil {
		bindErrs = append(bindErrs, fmt.Errorf("failed to bind web.metrics_path flag: %w", err))
	}
	if err := viper.BindPFlag("ldap.server", rootCmd.Flags().Lookup("ldap.server")); err != nil {
		bindErrs = append(bindErrs, fmt.Errorf("failed to bind ldap.server flag: %w", err))
	}
	if err := viper.BindPFlag("ldap.bind_dn", rootCmd.Flags().Lookup("ldap.bind-dn")); err != nil {
		bindErrs = append(bindErrs, fmt.Errorf("failed to bind ldap.bind_dn flag: %w", err))
	}
	if err := viper.BindPFlag("ldap.bind_password", rootCmd.Flags().Lookup("ldap.bind-password")); err != nil {
		bindErrs = append(bindErrs, fmt.Errorf("failed to bind ldap.bind_password flag: %w", err))
	}
	if err := viper.BindPFlag("ldap.timeout", rootCmd.Flags().Lookup("ldap.timeout")); err != nil {
		bindErrs = append(bindErrs, fmt.Errorf("failed to bind ldap.timeout flag: %w", err))
	}
	if err := viper.BindPFlag("ldap.start_tls", rootCmd.Flags().Lookup("ldap.start-tls")); err != nil {
		bindErrs = append(bindErrs, fmt.Errorf("failed to bind ldap.start_tls flag: %w", err))
	}
	if err := viper.BindPFlag("ldap.insecure_skip_verify", rootCmd.Flags().Lookup("ldap.insecure-skip-verify")); err != nil {
		bindErrs = append(bindErrs, fmt.Errorf("failed to bind ldap.insecure_skip_verify flag: %w", err))
	}
	if err := viper.BindPFlag("log.level", rootCmd.Flags().Lookup("log.level")); err != nil {
		bindErrs = append(bindErrs, fmt.Errorf("failed to bind log.level flag: %w", err))
	}
	if err := viper.BindPFlag("log.format", rootCmd.Flags().Lookup("log.format")); err != nil {
		bindErrs = append(bindErrs, fmt.Errorf("failed to bind log.format flag: %w", err))
	}
	
	// Handle binding errors gracefully
	if len(bindErrs) > 0 {
		for _, err := range bindErrs {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
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
