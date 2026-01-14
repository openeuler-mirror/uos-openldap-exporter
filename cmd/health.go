package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"gitee.com/openeuler/uos-openldap-exporter/internal/collector"
	"gitee.com/openeuler/uos-openldap-exporter/internal/config"
	"gitee.com/openeuler/uos-openldap-exporter/internal/logger"
)

// healthCmd represents the health command
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Perform health check against LDAP server",
	Long:  `Performs detailed health check against the LDAP server to diagnose connectivity issues.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgFile, _ := cmd.Flags().GetString("config.file")
		if cfgFile == "" {
			cfgFile = viper.GetString("config.file")
		}
		
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

		// Run the enhanced health check
		status, details := collector.EnhancedCheckLDAPHealth(&cfg.LDAP, log)
		if status {
			fmt.Println("✓ LDAP Health Check: SUCCESS")
			if details != "" {
				fmt.Printf("Details: %s\n", details)
			}
		} else {
			fmt.Println("✗ LDAP Health Check: FAILED")
			if details != "" {
				fmt.Printf("Error: %s\n", details)
			}
			os.Exit(1)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(healthCmd)

	// Allow overriding configuration from flags
	healthCmd.Flags().String("config.file", "", "Path to config file")
	healthCmd.Flags().String("ldap.server", "", "LDAP server URL (e.g., ldap://localhost:389)")
	healthCmd.Flags().String("ldap.bind-dn", "", "Bind DN for authentication")
	healthCmd.Flags().String("ldap.bind-password", "", "Bind password")
	healthCmd.Flags().Bool("ldap.start-tls", false, "Enable StartTLS")
	healthCmd.Flags().Bool("ldap.insecure-skip-verify", false, "Skip LDAP server certificate verification (NOT recommended for production)")

	// Bind flags to viper
	viper.BindPFlag("config.file", healthCmd.Flags().Lookup("config.file"))
	viper.BindPFlag("ldap.server", healthCmd.Flags().Lookup("ldap.server"))
	viper.BindPFlag("ldap.bind_dn", healthCmd.Flags().Lookup("ldap.bind-dn"))
	viper.BindPFlag("ldap.bind_password", healthCmd.Flags().Lookup("ldap.bind-password"))
	viper.BindPFlag("ldap.start_tls", healthCmd.Flags().Lookup("ldap.start-tls"))
	viper.BindPFlag("ldap.insecure_skip_verify", healthCmd.Flags().Lookup("ldap.insecure-skip-verify"))
}