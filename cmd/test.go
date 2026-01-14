package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"gitee.com/openeuler/uos-openldap-exporter/internal/collector"
	"gitee.com/openeuler/uos-openldap-exporter/internal/config"
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Test LDAP connection and basic operations",
	Long: `Tests LDAP connection and performs basic operations to diagnose issues.
This includes connecting to the server, binding with credentials, and performing
basic search operations.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfgFile, _ := cmd.Flags().GetString("config.file")
		if cfgFile == "" {
			cfgFile = viper.GetString("config.file")
		}
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}


		client, err := collector.NewTestLDAPClient(&cfg.LDAP)
		if err != nil {
			return fmt.Errorf("failed to create LDAP client: %w", err)
		}

		fmt.Println("Testing LDAP connection...")
		
		// Test connection
		startTime := time.Now()
		conn, err := client.Connect()
		if err != nil {
			return fmt.Errorf("connection test failed: %w", err)
		}
		fmt.Printf("✓ Connection established in %v\n", time.Since(startTime))

		// Test bind
		startTime = time.Now()
		if err := client.Bind(conn); err != nil {
			return fmt.Errorf("bind test failed: %w", err)
		}
		fmt.Printf("✓ Bind successful in %v\n", time.Since(startTime))

		// Test basic search in cn=Monitor
		startTime = time.Now()
		entries, err := client.Search(conn, "cn=Monitor", "(objectClass=*)", []string{"dn"})
		if err != nil {
			fmt.Printf("⚠ Monitor search failed: %v\n", err)
		} else {
			fmt.Printf("✓ Monitor search successful in %v, found %d entries\n", time.Since(startTime), len(entries))
		}

		// Test custom searches if any are configured
		for i, search := range cfg.CustomSearches {
			startTime = time.Now()
			entries, err := client.Search(conn, search.BaseDN, search.Filter, []string{"dn"})
			if err != nil {
				fmt.Printf("⚠ Custom search [%d: %s] failed: %v\n", i, search.Name, err)
			} else {
				fmt.Printf("✓ Custom search [%d: %s] successful in %v, found %d entries\n", i, search.Name, time.Since(startTime), len(entries))
			}
		}

		fmt.Println("\n✓ All tests completed successfully!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(testCmd)

	// Use the shared cfgFile variable from root command
	testCmd.Flags().String("config.file", "", "Path to config file")
	testCmd.Flags().String("ldap.server", "", "LDAP server URL (e.g., ldap://localhost:389)")
	testCmd.Flags().String("ldap.bind-dn", "", "Bind DN for authentication")
	testCmd.Flags().String("ldap.bind-password", "", "Bind password")
	testCmd.Flags().Bool("ldap.start-tls", false, "Enable StartTLS")
	testCmd.Flags().Bool("ldap.insecure-skip-verify", false, "Skip LDAP server certificate verification (NOT recommended for production)")

	// 在init阶段就绑定到Viper，确保与其他命令保持一致
	viper.BindPFlag("config.file", testCmd.Flags().Lookup("config.file"))
	viper.BindPFlag("ldap.server", testCmd.Flags().Lookup("ldap.server"))
	viper.BindPFlag("ldap.bind_dn", testCmd.Flags().Lookup("ldap.bind-dn"))
	viper.BindPFlag("ldap.bind_password", testCmd.Flags().Lookup("ldap.bind-password"))
	viper.BindPFlag("ldap.start_tls", testCmd.Flags().Lookup("ldap.start-tls"))
	viper.BindPFlag("ldap.insecure_skip_verify", testCmd.Flags().Lookup("ldap.insecure-skip-verify"))
}