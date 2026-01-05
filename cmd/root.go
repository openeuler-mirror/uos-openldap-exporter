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

		log := logger.NewWithConfig(logger.Config{
			Level:      cfg.Log.Level,
			Format:     cfg.Log.Format,
			Output:     cfg.Log.Output,
			MaxSize:    cfg.Log.MaxSize,
			MaxAge:     cfg.Log.MaxAge,
			MaxBackups: cfg.Log.MaxBackups,
			LocalTime:  cfg.Log.LocalTime,
			Compress:   cfg.Log.Compress,
		})
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
var (
	// Define flags as package variables so they can be accessed from Execute
	webListenAddress       *string
	webMetricsPath         *string
	ldapServer             *string
	ldapBindDN             *string
	ldapBindPassword       *string
	ldapTimeout            *time.Duration
	ldapStartTLS           *bool
	ldapInsecureSkipVerify *bool
	logLevel               *string
	logFormat              *string
	logOutput              *string
	logMaxSize             *int
	logMaxAge              *int
	logMaxBackups          *int
	logLocalTime           *bool
	logCompress            *bool
)

func Execute() error {
	// Initialize configuration and bind flags before executing the command
	if err := initConfigAndFlags(); err != nil {
		return err
	}

	if err := rootCmd.Execute(); err != nil {
		return err
	}
	return nil
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config.file", "", "Path to config file")
	rootCmd.AddCommand(versionCmd)

	// Web flags - store references to flags for later binding
	webListenAddress = rootCmd.Flags().String("web.listen-address", ":9330", "Address to listen on")
	webMetricsPath = rootCmd.Flags().String("web.metrics-path", "/metrics", "Path under which to expose metrics")

	// LDAP flags
	ldapServer = rootCmd.Flags().String("ldap.server", "", "LDAP server URL (e.g., ldap://localhost:389)")
	ldapBindDN = rootCmd.Flags().String("ldap.bind-dn", "", "Bind DN for authentication")
	ldapBindPassword = rootCmd.Flags().String("ldap.bind-password", "", "Bind password")
	ldapTimeout = rootCmd.Flags().Duration("ldap.timeout", 10*time.Second, "LDAP connection timeout")
	ldapStartTLS = rootCmd.Flags().Bool("ldap.start-tls", false, "Enable StartTLS")
	ldapInsecureSkipVerify = rootCmd.Flags().Bool("ldap.insecure-skip-verify", false, "Skip LDAP server certificate verification (NOT recommended for production)")

	// Log flags
	logLevel = rootCmd.Flags().String("log.level", "info", "Log level (debug, info, warn, error)")
	logFormat = rootCmd.Flags().String("log.format", "text", "Log format (text, json)")
	logOutput = rootCmd.Flags().String("log.output", "stdout", "Log output file path (default stdout)")
	logMaxSize = rootCmd.Flags().Int("log.max-size", 100, "Maximum size in megabytes of the log file before it gets rotated")
	logMaxAge = rootCmd.Flags().Int("log.max-age", 30, "Maximum number of days to retain old log files")
	logMaxBackups = rootCmd.Flags().Int("log.max-backups", 3, "Maximum number of old log files to retain")
	logLocalTime = rootCmd.Flags().Bool("log.local-time", false, "Use local time for log timestamp instead of UTC")
	logCompress = rootCmd.Flags().Bool("log.compress", false, "Compress rotated log files")
}

// initConfigAndFlags initializes the configuration and binds the flags to viper
func initConfigAndFlags() error {
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
	if err := viper.BindPFlag("log.output", rootCmd.Flags().Lookup("log.output")); err != nil {
		bindErrs = append(bindErrs, fmt.Errorf("failed to bind log.output flag: %w", err))
	}
	if err := viper.BindPFlag("log.max_size", rootCmd.Flags().Lookup("log.max-size")); err != nil {
		bindErrs = append(bindErrs, fmt.Errorf("failed to bind log.max_size flag: %w", err))
	}
	if err := viper.BindPFlag("log.max_age", rootCmd.Flags().Lookup("log.max-age")); err != nil {
		bindErrs = append(bindErrs, fmt.Errorf("failed to bind log.max_age flag: %w", err))
	}
	if err := viper.BindPFlag("log.max_backups", rootCmd.Flags().Lookup("log.max-backups")); err != nil {
		bindErrs = append(bindErrs, fmt.Errorf("failed to bind log.max_backups flag: %w", err))
	}
	if err := viper.BindPFlag("log.local_time", rootCmd.Flags().Lookup("log.local-time")); err != nil {
		bindErrs = append(bindErrs, fmt.Errorf("failed to bind log.local_time flag: %w", err))
	}
	if err := viper.BindPFlag("log.compress", rootCmd.Flags().Lookup("log.compress")); err != nil {
		bindErrs = append(bindErrs, fmt.Errorf("failed to bind log.compress flag: %w", err))
	}

	// Handle binding errors
	if len(bindErrs) > 0 {
		for _, err := range bindErrs {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		return fmt.Errorf("encountered %d configuration binding errors", len(bindErrs))
	}

	return nil
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
