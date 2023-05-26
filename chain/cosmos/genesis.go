package cosmos

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/icza/dyno"
	"github.com/strangelove-ventures/interchaintest/v3/ibc"
)

func ModifyGenesisProposalTime(votingPeriod string, maxDepositPeriod string) func(ibc.ChainConfig, []byte) ([]byte, error) {
	return func(chainConfig ibc.ChainConfig, genbz []byte) ([]byte, error) {
		g := make(map[string]interface{})
		if err := json.Unmarshal(genbz, &g); err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis file: %w", err)
		}
		if err := dyno.Set(g, votingPeriod, "app_state", "gov", "voting_params", "voting_period"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		if err := dyno.Set(g, maxDepositPeriod, "app_state", "gov", "deposit_params", "max_deposit_period"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		if err := dyno.Set(g, chainConfig.Denom, "app_state", "gov", "deposit_params", "min_deposit", 0, "denom"); err != nil {
			return nil, fmt.Errorf("failed to set voting period in genesis json: %w", err)
		}
		out, err := json.Marshal(g)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal genesis bytes to json: %w", err)
		}
		return out, nil
	}
}

func PrintGenesis() func(ibc.ChainConfig, []byte) ([]byte, error) {
	return func(chainConfig ibc.ChainConfig, genbz []byte) ([]byte, error) {
		g := make(map[string]interface{})
		if err := json.Unmarshal(genbz, &g); err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis file: %w", err)
		}
		print("\n\n GAIA GENESIS \n\n")

		// pp := func(err error) {
		// 	json.NewEncoder(os.Stdout).Encode(g) // Output JSON
		// 	if err != nil {
		// 		fmt.Println("ERROR:", err)
		// 	}
		// }
		print(json.NewEncoder(os.Stdout).Encode(g))

		out, err := json.Marshal(g)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal genesis bytes to json: %w", err)
		}
		return out, nil
	}
}

func ModifyNeutronGenesis(
	soft_opt_out_threshold string,
	reward_denoms []string,
	provider_reward_denoms []string) func(ibc.ChainConfig, []byte) ([]byte, error) {
	return func(chainConfig ibc.ChainConfig, genbz []byte) ([]byte, error) {
		g := make(map[string]interface{})
		print("\n\n Modifying neutron genesis\n")
		if err := json.Unmarshal(genbz, &g); err != nil {
			return nil, fmt.Errorf("failed to unmarshal genesis file: %w", err)
		}

		if err := dyno.Set(g, soft_opt_out_threshold, "app_state", "ccvconsumer", "params", "soft_opt_out_threshold"); err != nil {
			return nil, fmt.Errorf("failed to set soft_opt_out_threshold in genesis json: %w", err)
		}

		if err := dyno.Set(g, reward_denoms, "app_state", "ccvconsumer", "params", "reward_denoms"); err != nil {
			return nil, fmt.Errorf("failed to set reward_denoms in genesis json: %w", err)
		}

		if err := dyno.Set(g, provider_reward_denoms, "app_state", "ccvconsumer", "params", "provider_reward_denoms"); err != nil {
			return nil, fmt.Errorf("failed to set provider_reward_denoms in genesis json: %w", err)
		}

		// faucetBalEntry := map[string]interface{}{
		// 	"address": "neutron1rfk2927e7vdu5dfdml5qn57kytn9cc2an6ghqs",
		// 	"coins": []map[string]interface{}{
		// 		{
		// 			"denom":  "untrn",
		// 			"amount": "1000000000000",
		// 		},
		// 	},
		// }
		// // fund faucet with some tokens
		// if err := dyno.Append(g, faucetBalEntry, "app_state", "bank", "balances"); err != nil {
		// 	return nil, fmt.Errorf("failed to set provider_reward_denoms in genesis json: %w", err)
		// }

		// // reflect added tokens to the max supply
		// totalSupplyEntry := []map[string]interface{}{
		// 	{
		// 		"denom":  "untrn",
		// 		"amount": "102000000000000",
		// 	},
		// }
		// if err := dyno.Set(g, totalSupplyEntry, "app_state", "bank", "supply"); err != nil {
		// 	return nil, fmt.Errorf("failed to update max supply")
		// }

		out, err := json.Marshal(g)

		print("\n\n NEUTRON GENESIS\n")
		print(json.NewEncoder(os.Stdout).Encode(g))

		if err != nil {
			return nil, fmt.Errorf("failed to marshal genesis bytes to json: %w", err)
		}
		return out, nil
	}
}
