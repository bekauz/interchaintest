package ibc_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	transfertypes "github.com/cosmos/ibc-go/v3/modules/apps/transfer/types"
	ibctest "github.com/strangelove-ventures/interchaintest/v3"
	"github.com/strangelove-ventures/interchaintest/v3/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v3/ibc"
	"github.com/strangelove-ventures/interchaintest/v3/relayer"
	"github.com/strangelove-ventures/interchaintest/v3/relayer/rly"
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

	// neutronValidators := int(0)
	// neutronFullnodes := int(0)
	var reward_denoms [1]string
	var provider_reward_denoms [1]string

	reward_denoms[0] = "untrn"
	provider_reward_denoms[0] = "uatom"
	// Chain Factory
	cf := ibctest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*ibctest.ChainSpec{
		// {Name: "gaia", Version: "v9.1.0", ChainConfig: ibc.ChainConfig{GasAdjustment: 1.5}},
		{Name: "gaia", Version: "v9.1.0", ChainConfig: ibc.ChainConfig{
			ModifyGenesis: cosmos.PrintGenesis(),
			GasPrices:     "0.0atom",
		}},
		// {Name: "neutron", Version: "v1.0.1", ChainConfig: ibc.ChainConfig{
		// 	ModifyGenesis: cosmos.ModifyNeutronGenesis("0.05", reward_denoms[:], provider_reward_denoms[:]),
		// }},
		{
			ChainConfig: ibc.ChainConfig{
				Type:    "cosmos",
				Name:    "neutron",
				ChainID: "neutron-2",
				Images: []ibc.DockerImage{
					{
						Repository: "neutron-node",
						Version:    "latest",
					},
				},
				Bin:            "neutrond",
				Bech32Prefix:   "neutron",
				Denom:          "untrn",
				GasPrices:      "0.0untrn",
				GasAdjustment:  1.3,
				TrustingPeriod: "1197504s",
				NoHostMount:    false,
				ModifyGenesis:  cosmos.ModifyNeutronGenesis("0.05", reward_denoms[:], provider_reward_denoms[:]),
			},
		},
		// {Name: "stride", Version: "v9.0.0"},
	})

	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)
	provider, consumer := chains[0], chains[1]
	// provider, consumer, stride := chains[0], chains[1], chains[2]

	// Relayer Factory
	client, network := ibctest.DockerSetup(t)
	r := ibctest.NewBuiltinRelayerFactory(
		ibc.CosmosRly,
		zaptest.NewLogger(t),
		relayer.CustomDockerImage("ghcr.io/cosmos/relayer", "v2.3.1", rly.RlyDefaultUidGid),
		relayer.RelayerOptionExtraStartFlags{Flags: []string{"-d", "--log-format", "console"}},
	).Build(t, client, network)

	// Prep Interchain
	const ibcPath = "ics-path"
	ic := ibctest.NewInterchain().
		AddChain(provider).
		AddChain(consumer).
		// AddChain(stride).
		AddRelayer(r, "relayer").
		AddProviderConsumerLink(ibctest.ProviderConsumerLink{
			Provider: provider,
			Consumer: consumer,
			Relayer:  r,
			Path:     ibcPath,
		})
		// AddLink(ibctest.InterchainLink{
		// 	Chain1:  consumer,
		// 	Chain2:  stride,
		// 	Relayer: r,
		// 	Path:    "neutron-stride-path",
		// })
		// AddLink(ibctest.InterchainLink{
		// 	Chain1:  provider,
		// 	Chain2:  stride,
		// 	Relayer: r,
		// 	Path:    "gaia-stride-path",
		// })

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

	// Create and Fund User Wallets on gaia, neutron, and stride
	fundAmount := int64(10_000_000)
	users := ibctest.GetAndFundTestUsers(t, ctx, "default", fundAmount, provider, consumer)

	gaiaUser := users[0]
	neutronUser := users[1]
	// strideUser := users[2]
	// Wait a few blocks for user accounts to be created on chain.
	err = testutil.WaitForBlocks(ctx, 5, provider, consumer)
	require.NoError(t, err)

	gaiaUserBalInitial, err := provider.GetBalance(
		ctx,
		gaiaUser.Bech32Address(provider.Config().Bech32Prefix),
		provider.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, fundAmount, gaiaUserBalInitial)

	// not funding neutron
	neutronUserBalInitial, err := consumer.GetBalance(
		ctx,
		neutronUser.Bech32Address(consumer.Config().Bech32Prefix),
		consumer.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, int64(0), neutronUserBalInitial)

	// Get Channel ID
	// gaiaChannelInfo, err := r.GetChannels(ctx, eRep, provider.Config().ChainID)
	// require.NoError(t, err)
	// gaiaChannelID := gaiaChannelInfo[0].ChannelID

	// neutronChannelInfo, err := r.GetChannels(ctx, eRep, consumer.Config().ChainID)
	// require.NoError(t, err)
	// neutronChannelID := neutronChannelInfo[0].ChannelID

	gaiaNeutronChannel, err := ibc.GetTransferChannel(
		ctx,
		r,
		eRep,
		provider.Config().ChainID,
		consumer.Config().ChainID)
	require.NoError(t, err)

	amountToSend := int64(500_000)
	dstAddress := neutronUser.Bech32Address(consumer.Config().Bech32Prefix)
	transfer := ibc.WalletAmount{
		Address: dstAddress,
		Denom:   provider.Config().Denom,
		Amount:  amountToSend,
	}
	tx, err := provider.SendIBCTransfer(
		ctx,
		gaiaNeutronChannel.ChannelID,
		gaiaUser.GetKeyName(),
		transfer,
		ibc.TransferOptions{})
	require.NoError(t, err)
	require.NoError(t, tx.Validate())

	// relay packets and acknoledgments
	require.NoError(t, r.FlushPackets(ctx, eRep, ibcPath, gaiaNeutronChannel.ChannelID))
	require.NoError(t, r.FlushAcknowledgements(ctx, eRep, ibcPath, gaiaNeutronChannel.ChannelID))

	// test source wallet has decreased funds
	expectedBal := gaiaUserBalInitial - amountToSend
	gaiaUserBalNew, err := provider.GetBalance(
		ctx,
		gaiaUser.Bech32Address(provider.Config().Bech32Prefix),
		provider.Config().Denom)
	require.NoError(t, err)
	require.Equal(t, expectedBal, gaiaUserBalNew)

	// Trace IBC Denom
	srcDenomTrace := transfertypes.ParseDenomTrace(
		transfertypes.GetPrefixedDenom("transfer", gaiaNeutronChannel.Counterparty.ChannelID, provider.Config().Denom))
	dstIbcDenom := srcDenomTrace.IBCDenom()

	// Test destination wallet has increased funds
	neutronUserBalNew, err := consumer.GetBalance(
		ctx,
		neutronUser.Bech32Address(consumer.Config().Bech32Prefix),
		dstIbcDenom)
	require.NoError(t, err)
	require.Equal(t, amountToSend, neutronUserBalNew)
}
