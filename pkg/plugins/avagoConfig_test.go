// Copyright (C) 2022, Lux Partners Limited, All rights reserved.
// See the file LICENSE for licensing terms.

package plugins

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/luxdefi/cli/internal/testutils"
	"github.com/luxdefi/cli/pkg/constants"
	"github.com/luxdefi/cli/pkg/models"
	"github.com/luxdefi/cli/pkg/ux"

	"github.com/luxdefi/node/utils/logging"
	"github.com/stretchr/testify/require"
)

const (
	subnetName1 = "TEST_subnet"
	subnetName2 = "TEST_copied_subnet"

	subnetID = "testSubNet"
)

var network = models.Network{ID: 67443}

// testing backward compatibility
func TestEditConfigFileWithOldPattern(t *testing.T) {
	ux.NewUserLog(logging.NoLog{}, io.Discard)

	require := require.New(t)

	ap := testutils.SetupTestInTempDir(t)

	genesisBytes := []byte("genesis")
	err := ap.WriteGenesisFile(subnetName1, genesisBytes)
	require.NoError(err)

	configFile := constants.NodeFileName

	// Create ConfigFile
	tmpDir := os.TempDir()
	configPath := filepath.Join(tmpDir, configFile)
	defer os.Remove(configPath)

	// testing backward compatibility
	configBytes := []byte("{\"whitelisted-subnets\": \"subNetId000\"}")
	err = os.MkdirAll(filepath.Dir(configPath), constants.DefaultPerms755)
	require.NoError(err)
	err = os.WriteFile(configPath, configBytes, 0o600)
	require.NoError(err)

	err = EditConfigFile(ap, subnetID, network, configPath, true, "")
	require.NoError(err)

	fileBytes, err := os.ReadFile(configPath)
	require.NoError(err)

	var luxdConfig map[string]interface{}
	err = json.Unmarshal(fileBytes, &luxdConfig)
	require.NoError(err)

	require.Equal("subNetId000,testSubNet", luxdConfig["track-subnets"])

	// ensure that the old setting has been deleted
	require.Equal(nil, luxdConfig["whitelisted-subnets"])
}

// testing backward compatibility
func TestEditConfigFileWithNewPattern(t *testing.T) {
	ux.NewUserLog(logging.NoLog{}, io.Discard)

	require := require.New(t)

	ap := testutils.SetupTestInTempDir(t)

	genesisBytes := []byte("genesis")
	err := ap.WriteGenesisFile(subnetName1, genesisBytes)
	require.NoError(err)

	configFile := constants.NodeFileName

	// Create ConfigFile
	tmpDir := os.TempDir()
	configPath := filepath.Join(tmpDir, configFile)
	defer os.Remove(configPath)

	// testing backward compatibility
	configBytes := []byte("{\"track-subnets\": \"subNetId000\"}")
	err = os.MkdirAll(filepath.Dir(configPath), constants.DefaultPerms755)
	require.NoError(err)
	err = os.WriteFile(configPath, configBytes, 0o600)
	require.NoError(err)

	err = EditConfigFile(ap, subnetID, network, configPath, true, "")
	require.NoError(err)

	fileBytes, err := os.ReadFile(configPath)
	require.NoError(err)

	var luxdConfig map[string]interface{}
	err = json.Unmarshal(fileBytes, &luxdConfig)
	require.NoError(err)

	require.Equal("subNetId000,testSubNet", luxdConfig["track-subnets"])

	// ensure that the old setting wont be applied at all
	require.Equal(nil, luxdConfig["whitelisted-subnets"])
}

func TestEditConfigFileWithNoSettings(t *testing.T) {
	ux.NewUserLog(logging.NoLog{}, io.Discard)

	require := require.New(t)

	ap := testutils.SetupTestInTempDir(t)

	genesisBytes := []byte("genesis")
	err := ap.WriteGenesisFile(subnetName1, genesisBytes)
	require.NoError(err)

	configFile := constants.NodeFileName

	// Create ConfigFile
	tmpDir := os.TempDir()
	configPath := filepath.Join(tmpDir, configFile)
	defer os.Remove(configPath)

	// testing when no setting for tracked subnets exists
	configBytes := []byte("{\"networkId\": \"5\"}")
	err = os.MkdirAll(filepath.Dir(configPath), constants.DefaultPerms755)
	require.NoError(err)
	err = os.WriteFile(configPath, configBytes, 0o600)
	require.NoError(err)

	err = EditConfigFile(ap, subnetID, network, configPath, true, "")
	require.NoError(err)

	fileBytes, err := os.ReadFile(configPath)
	require.NoError(err)

	var luxdConfig map[string]interface{}
	err = json.Unmarshal(fileBytes, &luxdConfig)
	require.NoError(err)

	require.Equal("testSubNet", luxdConfig["track-subnets"])

	// ensure that the old setting wont be applied at all
	require.Equal(nil, luxdConfig["whitelisted-subnets"])
}
