package ibc_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	ibctest "github.com/strangelove-ventures/interchaintest/v3"
	"github.com/strangelove-ventures/interchaintest/v3/ibc"
	"github.com/strangelove-ventures/interchaintest/v3/testreporter"
	"github.com/strangelove-ventures/interchaintest/v3/testutil"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// This tests Cosmos Interchain Security, spinning up a provider and a single consumer chain.
func TestICS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	t.Parallel()

	ctx := context.Background()

	// Chain Factory
	cf := ibctest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*ibctest.ChainSpec{
		{Name: "gaia", Version: "v9.0.0-rc1", ChainConfig: ibc.ChainConfig{GasAdjustment: 1.5}},
		// {Name: "neutron", Version: "v1.0.0-rc1"}
		{ChainConfig: ibc.ChainConfig{
			Name:    "neutron",
			ChainID: "test-neutron",
			Images: []ibc.DockerImage{
				{
					Repository: "neutron",    // FOR LOCAL IMAGE USE: Docker Image Name
					Version:    "v1.0.0-rc1", // FOR LOCAL IMAGE USE: Docker Image Tag
				},
			},
			Bin:            "neutrond",
			Bech32Prefix:   "neutron",
			Denom:          "untrn",
			GasPrices:      "0.0untrn",
			GasAdjustment:  1.3,
			TrustingPeriod: "508h",
			NoHostMount:    false},
		},
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)
	provider, consumer := chains[0], chains[1]

	// Relayer Factory
	client, network := ibctest.DockerSetup(t)
	// r := ibctest.NewBuiltinRelayerFactory(
	// 	ibc.Hermes,
	// 	zaptest.NewLogger(t),
	// 	relayer.CustomDockerImage("ghcr.io/cosmos/relayer", "andrew-paths_update", rly.RlyDefaultUidGid),
	// ).Build(t, client, network)
	r := ibctest.NewBuiltinRelayerFactory(
		ibc.Hermes,
		zaptest.NewLogger(t)).Build(t, client, network)

	// Prep Interchain
	const ibcPath = "ics-path"
	ic := ibctest.NewInterchain().
		AddChain(provider).
		AddChain(consumer).
		AddRelayer(r, "relayer").
		AddProviderConsumerLink(ibctest.ProviderConsumerLink{
			Provider: provider,
			Consumer: consumer,
			Relayer:  r,
			Path:     ibcPath,
		})

	// Log location
	f, err := ibctest.CreateLogFile(fmt.Sprintf("%d.json", time.Now().Unix()))
	require.NoError(t, err)
	// Reporter/logs
	rep := testreporter.NewReporter(f)
	eRep := rep.RelayerExecReporter(t)

	// Build interchain
	err = ic.Build(ctx, eRep, ibctest.InterchainBuildOptions{
		TestName:          t.Name(),
		Client:            client,
		NetworkID:         network,
		BlockDatabaseFile: ibctest.DefaultBlockDatabaseFilepath(),

		SkipPathCreation: false,
	})
	require.NoError(t, err, "failed to build interchain")

	err = testutil.WaitForBlocks(ctx, 10, provider, consumer)
	require.NoError(t, err, "failed to wait for blocks")
}
