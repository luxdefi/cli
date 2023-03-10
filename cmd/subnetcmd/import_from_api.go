// Copyright (C) 2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.
package subnetcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/luxdefi/cli/pkg/constants"
	"github.com/luxdefi/cli/pkg/models"
	"github.com/luxdefi/cli/pkg/ux"
	"github.com/luxdefi/cli/pkg/vm"
	"github.com/luxdefi/node/api/info"
	"github.com/luxdefi/node/ids"
	"github.com/luxdefi/node/utils/rpc"
	"github.com/luxdefi/node/vms/platformvm"
	"github.com/luxdefi/coreth/core"
	"github.com/luxdefi/spacesvm/chain"
	"github.com/spf13/cobra"
)

var (
	genesisFilePath string
	subnetIDstr     string
	nodeURL         string
)

// avalanche subnet import
func newImportFromNetworkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "public [subnetPath]",
		Short:        "Import an existing subnet config from running subnets on a public network",
		RunE:         importRunningSubnet,
		SilenceUsage: true,
		Args:         cobra.MaximumNArgs(1),
		Long: `The subnet import public command will import a subnet configuration from a running network.

The genesis file should be available from the disk for this to work. 
By default, an imported subnet will not overwrite an existing subnet with the same name. 
To allow overwrites, provide the --force flag.`,
	}

	cmd.Flags().StringVar(&nodeURL, "node-url", "", "[optional] URL of an already running subnet validator")

	cmd.Flags().BoolVar(&deployTestnet, "fuji", false, "import from `fuji` (alias for `testnet`)")
	cmd.Flags().BoolVar(&deployTestnet, "testnet", false, "import from `testnet` (alias for `fuji`)")
	cmd.Flags().BoolVar(&deployMainnet, "mainnet", false, "import from `mainnet`")
	cmd.Flags().BoolVar(&useSubnetEvm, "evm", false, "import a subnet-evm")
	cmd.Flags().BoolVar(&useSpacesVM, "spacesvm", false, "use the SpacesVM as the base template")
	cmd.Flags().BoolVar(&useCustom, "custom", false, "use a custom VM template")
	cmd.Flags().BoolVarP(
		&overwriteImport,
		"force",
		"f",
		false,
		"overwrite the existing configuration if one exists",
	)
	cmd.Flags().StringVar(
		&genesisFilePath,
		"genesis-file-path",
		"",
		"path to the genesis file",
	)
	cmd.Flags().StringVar(
		&subnetIDstr,
		"subnet-id",
		"",
		"the subnet ID",
	)
	return cmd
}

func importRunningSubnet(*cobra.Command, []string) error {
	var err error

	var network models.Network
	switch {
	case deployTestnet:
		network = models.Fuji
	case deployMainnet:
		network = models.Mainnet
	}

	if network == models.Undefined {
		networkStr, err := app.Prompt.CaptureList(
			"Choose a network to import from",
			[]string{models.Fuji.String(), models.Mainnet.String()},
		)
		if err != nil {
			return err
		}
		network = models.NetworkFromString(networkStr)
	}

	if genesisFilePath == "" {
		genesisFilePath, err = app.Prompt.CaptureExistingFilepath("Provide the path to the genesis file")
		if err != nil {
			return err
		}
	}

	var reply *info.GetNodeVersionReply

	if nodeURL == "" {
		yes, err := app.Prompt.CaptureYesNo("Have nodes already been deployed to this subnet?")
		if err != nil {
			return err
		}
		if yes {
			nodeURL, err = app.Prompt.CaptureString(
				"Please provide an API URL of such a node so we can query its VM version (e.g. http://111.22.33.44:5555)")
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), constants.RequestTimeout)
			defer cancel()
			infoAPI := info.NewClient(nodeURL)
			options := []rpc.Option{}
			reply, err = infoAPI.GetNodeVersion(ctx, options...)
			if err != nil {
				return fmt.Errorf("failed to query node - is it running and reachable? %w", err)
			}
		}
	}

	var subnetID ids.ID
	if subnetIDstr == "" {
		subnetID, err = app.Prompt.CaptureID("What is the ID of the subnet?")
		if err != nil {
			return err
		}
	} else {
		subnetID, err = ids.FromString(subnetIDstr)
		if err != nil {
			return err
		}
	}

	var pubAPI string
	switch network {
	case models.Fuji:
		pubAPI = constants.FujiAPIEndpoint
	case models.Mainnet:
		pubAPI = constants.MainnetAPIEndpoint
	}
	client := platformvm.NewClient(pubAPI)
	ctx, cancel := context.WithTimeout(context.Background(), constants.RequestTimeout)
	defer cancel()
	options := []rpc.Option{}

	ux.Logger.PrintToUser("Getting information from the %s network...", network.String())

	chains, err := client.GetBlockchains(ctx, options...)
	if err != nil {
		return err
	}

	var (
		blockchainID, vmID ids.ID
		subnetName         string
	)

	for _, ch := range chains {
		// NOTE: This supports only one chain per subnet
		if ch.SubnetID == subnetID {
			blockchainID = ch.ID
			vmID = ch.VMID
			subnetName = ch.Name
			break
		}
	}

	if blockchainID == ids.Empty || vmID == ids.Empty {
		return fmt.Errorf("subnet ID %s not found on this network", subnetIDstr)
	}

	ux.Logger.PrintToUser("Retrieved information. BlockchainID: %s, Name: %s, VMID: %s",
		blockchainID.String(),
		subnetName,
		vmID.String(),
	)
	// TODO: it's probably possible to deploy VMs with the same name on a public network
	// In this case, an import could clash because the tool supports unique names only

	genBytes, err := os.ReadFile(genesisFilePath)
	if err != nil {
		return err
	}

	if err = app.WriteGenesisFile(subnetName, genBytes); err != nil {
		return err
	}

	vmType := getVMFromFlag()
	if vmType == "" {
		subnetTypeStr, err := app.Prompt.CaptureList(
			"What's this VM's type?",
			[]string{models.SubnetEvm, models.SpacesVM, models.CustomVM},
		)
		if err != nil {
			return err
		}
		vmType = models.VMTypeFromString(subnetTypeStr)
	}

	vmIDstr := vmID.String()

	sc := &models.Sidecar{
		Name: subnetName,
		VM:   vmType,
		Networks: map[string]models.NetworkData{
			network.String(): {
				SubnetID:     subnetID,
				BlockchainID: blockchainID,
			},
		},
		Subnet:       subnetName,
		Version:      constants.SidecarVersion,
		TokenName:    constants.DefaultTokenName,
		ImportedVMID: vmIDstr,
		// signals that the VMID wasn't derived from the subnet name but through import
		ImportedFromAPM: true,
	}

	var versions []string

	if reply != nil {
		// a node was queried
		for _, v := range reply.VMVersions {
			if v == vmIDstr {
				sc.VMVersion = v
				break
			}
		}
		sc.RPCVersion = int(reply.RPCProtocolVersion)
	} else {
		// no node was queried, ask the user
		switch vmType {
		case models.SubnetEvm:
			versions, err = app.Downloader.GetAllReleasesForRepo(constants.AvaLabsOrg, constants.SubnetEVMRepoName)
			if err != nil {
				return err
			}
			sc.VMVersion, err = app.Prompt.CaptureList("Pick the version for this VM", versions)
		case models.SpacesVM:
			versions, err = app.Downloader.GetAllReleasesForRepo(constants.AvaLabsOrg, constants.SpacesVMRepoName)
			if err != nil {
				return err
			}
			sc.VMVersion, err = app.Prompt.CaptureList("Pick the version for this VM", versions)
		case models.CustomVM:
			return fmt.Errorf("importing custom VMs is not yet implemented, but will be available soon")
		default:
			return fmt.Errorf("unexpected VM type: %v", vmType)
		}
		if err != nil {
			return err
		}
		sc.RPCVersion, err = vm.GetRPCProtocolVersion(app, vmType, sc.VMVersion)
		if err != nil {
			return fmt.Errorf("failed getting RPCVersion for VM type %s with version %s", vmType, sc.VMVersion)
		}
	}

	switch vmType {
	case models.SubnetEvm:
		var genesis core.Genesis
		if err := json.Unmarshal(genBytes, &genesis); err != nil {
			return err
		}
		sc.ChainID = genesis.Config.ChainID.String()
	case models.SpacesVM:
		// for spacesvm just make sure it's valid
		var genesis chain.Genesis
		if err := json.Unmarshal(genBytes, &genesis); err != nil {
			return err
		}
	}

	if err := app.CreateSidecar(sc); err != nil {
		return fmt.Errorf("failed creating the sidecar for import: %w", err)
	}

	ux.Logger.PrintToUser("Subnet %s imported successfully", sc.Name)

	return nil
}
