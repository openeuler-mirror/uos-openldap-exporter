package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"gitee.com/openeuler/uos-openldap-exporter/internal/config"
	"gitee.com/openeuler/uos-openldap-exporter/internal/logger"
)

// validateCmd represents the validate command
var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate configuration file",
	Long:  `Validates the configuration file without starting the exporter.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgFile, _ := cmd.Flags().GetString("config.file")
		if cfgFile == "" {
			cfgFile = viper.GetString("config.file")
		}
		
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
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

		// If we reach here, the config loaded and validated successfully
		fmt.Println("✓ Configuration is valid")
		log.Info("Configuration validation completed successfully")
		
		// Print some key configuration values (without sensitive data)
		fmt.Printf("LDAP Server: %s\n", cfg.LDAP.Server)
		fmt.Printf("Listen Address: %s\n", cfg.Web.ListenAddress)
		fmt.Printf("Metrics Path: %s\n", cfg.Web.MetricsPath)
		fmt.Printf("Log Level: %s\n", cfg.Log.Level)
		fmt.Printf("Log Format: %s\n", cfg.Log.Format)
		fmt.Printf("Number of Custom Searches: %d\n", len(cfg.CustomSearches))
		fmt.Printf("Enabled Plugins: %v\n", cfg.Plugins.Enabled)
		
		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)

	// Add flags
	validateCmd.Flags().String("config.file", "", "Path to config file")
	
	// Bind flags to viper
	viper.BindPFlag("config.file", validateCmd.Flags().Lookup("config.file"))
}