// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package commands

import (
	"fmt"
	"os/exec"

	"github.com/luxdefi/cli/tests/e2e/utils"
)

/* #nosec G204 */
func UpgradeVMConfig(subnetName string, targetVersion string) (string, error) {
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		UpgradeCmd,
		"vm",
		subnetName,
		"--config",
		"--version",
		targetVersion,
	)

	output, err := cmd.Output()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	return string(output), err
}

func UpgradeVMPublic(subnetName string, targetVersion string, pluginDir string) (string, error) {
	cmd := exec.Command(
		CLIBinary,
		SubnetCmd,
		UpgradeCmd,
		"vm",
		subnetName,
		"--fuji",
		"--version",
		targetVersion,
		"--plugin-dir",
		pluginDir,
	)

	output, err := cmd.Output()
	if err != nil {
		fmt.Println(cmd.String())
		fmt.Println(string(output))
		utils.PrintStdErr(err)
	}
	return string(output), err
}
