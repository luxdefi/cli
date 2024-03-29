// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.
package nodecmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/luxdefi/cli/pkg/ansible"
	"github.com/luxdefi/cli/pkg/models"
	"github.com/luxdefi/cli/pkg/utils"
	"github.com/luxdefi/cli/pkg/ux"
	"github.com/spf13/cobra"
)

func newSSHCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssh [clusterName] [cmd]",
		Short: "(ALPHA Warning) Execute ssh command on node(s)",
		Long: `(ALPHA Warning) This command is currently in experimental mode.

The node ssh command execute a given command using ssh on all nodes in the cluster.
If no command is given, just prints the ssh cmdLine to be used to connect to each node.
`,
		SilenceUsage: true,
		Args:         cobra.MinimumNArgs(0),
		RunE:         sshNode,
	}

	return cmd
}

func sshNode(_ *cobra.Command, args []string) error {
	if len(args) == 0 {
		var err error
		clustersConfig := models.ClustersConfig{}
		if app.ClustersConfigExists() {
			clustersConfig, err = app.LoadClustersConfig()
			if err != nil {
				return err
			}
		}
		if len(clustersConfig.Clusters) == 0 {
			ux.Logger.PrintToUser("There are no clusters defined.")
			return nil
		}
		for clusterName, clusterConfig := range clustersConfig.Clusters {
			ux.Logger.PrintToUser("Cluster %q (%s)", clusterName, clusterConfig.Network.Name())
			if err := sshCluster([]string{clusterName}, "  "); err != nil {
				return err
			}
			ux.Logger.PrintToUser("")
		}
		return nil
	}
	return sshCluster(args, "")
}

func sshCluster(args []string, indent string) error {
	clusterName := args[0]
	if err := checkCluster(clusterName); err != nil {
		return err
	}
	hosts, err := ansible.GetInventoryFromAnsibleInventoryFile(app.GetAnsibleInventoryDirPath(clusterName))
	if err != nil {
		return err
	}
	for _, host := range hosts {
		cmdLine := fmt.Sprintf("%s %s", utils.GetSSHConnectionString(host.IP, host.SSHPrivateKeyPath), strings.Join(args[1:], " "))
		ux.Logger.PrintToUser("%s[%s] %s", indent, host.GetCloudID(), cmdLine)
		if len(args) > 1 {
			splitCmdLine := strings.Split(cmdLine, " ")
			cmd := exec.Command(splitCmdLine[0], splitCmdLine[1:]...) //nolint: gosec
			cmd.Env = os.Environ()
			_, _ = utils.SetupRealtimeCLIOutput(cmd, true, true)
			err = cmd.Run()
			if err != nil {
				ux.Logger.PrintToUser("Error: %s", err)
			}
			ux.Logger.PrintToUser("")
		}
	}
	return nil
}
