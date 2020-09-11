package app

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli"
)

const (
	timeoutFlag    = "timeout"
	defaultTimeout = 5 * time.Second
)

// AddressesFromContext returns a list of addresses from given and cli flag.
// It will return an error if any provided address is invalid.
func AddressesFromContext(c *cli.Context, flag string) ([]common.Address, error) {
	var (
		results []common.Address
		addrs   = c.StringSlice(flag)
	)
	if len(addrs) == 0 {
		return nil, fmt.Errorf("%s is empty", flag)
	}

	for _, addr := range addrs {
		if !common.IsHexAddress(addr) {
			return nil, fmt.Errorf("flag %s: invalid ethereum address %s", flag, addr)
		}
		results = append(results, common.HexToAddress(addr))
	}

	return results, nil
}

// NewTimeoutFlag return client flag to config timeout
func NewTimeoutFlag() cli.Flag {
	return cli.DurationFlag{
		Name:   timeoutFlag,
		Usage:  "provide timeout for request",
		EnvVar: "TIMEOUT",
		Value:  defaultTimeout,
	}
}

// TimeoutFromContext return timeout from client configures
func TimeoutFromContext(c *cli.Context) time.Duration {
	return c.Duration(timeoutFlag)
}
