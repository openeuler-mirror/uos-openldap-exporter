package main

import (
	"os"



	"gitee.com/openeuler/uos-openldap-exporter/cmd"

)

func main() {


	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
