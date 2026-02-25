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

// bindFlag binds a single flag to viper and handles errors
func bindFlag(viperKey, flagName string, bindErrs *[]error) {
	if err := viper.BindPFlag(viperKey, rootCmd.Flags().Lookup(flagName)); err != nil {
		*bindErrs = append(*bindErrs, fmt.Errorf("failed to bind %s flag: %w", viperKey, err))
	}
}

// initConfigAndFlags initializes the configuration and binds the flags to viper
func initConfigAndFlags() error {
	// Bind viper flags
	bindErrs := []error{}

	// Define flag mappings for cleaner binding
	flagMappings := []struct {
		viperKey string
		flagName string
	}{
		{"web.listen_address", "web.listen-address"},
		{"web.metrics_path", "web.metrics-path"},
		{"ldap.server", "ldap.server"},
		{"ldap.bind_dn", "ldap.bind-dn"},
		{"ldap.bind_password", "ldap.bind-password"},
		{"ldap.timeout", "ldap.timeout"},
		{"ldap.start_tls", "ldap.start-tls"},
		{"ldap.insecure_skip_verify", "ldap.insecure-skip-verify"},
		{"log.level", "log.level"},
		{"log.format", "log.format"},
		{"log.output", "log.output"},
		{"log.max_size", "log.max-size"},
		{"log.max_age", "log.max-age"},
		{"log.max_backups", "log.max-backups"},
		{"log.local_time", "log.local-time"},
		{"log.compress", "log.compress"},
	}

	// Bind all flags using the mapping
	for _, mapping := range flagMappings {
		bindFlag(mapping.viperKey, mapping.flagName, &bindErrs)
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
	// Set environment variable prefix and automatic env first
	viper.SetEnvPrefix("OPENLDAP_EXPORTER")
	viper.AutomaticEnv() // read in environment variables that match

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Default config file name
		viper.SetConfigName("config")
	}

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
