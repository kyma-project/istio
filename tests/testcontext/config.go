package testcontext

import (
	"fmt"
	"time"

	"github.com/avast/retry-go"
	"github.com/caarlos0/env/v11"
)

type Config struct {
	ClientTimeout time.Duration `env:"TEST_CLIENT_TIMEOUT envDefault=10s"`
	ReqTimeout    time.Duration `env:"TEST_REQUEST_TIMEOUT envDefault=300s"`
	ReqDelay      time.Duration `env:"TEST_REQUEST_DELAY envDefault=5s"`
}

var (
	retryOpts []retry.Option
)

func GetRetryOpts() []retry.Option {
	if retryOpts == nil {
		config, err := env.ParseAs[Config]()
		if err != nil {
			panic(fmt.Sprintf("Unable to setup test config: %v", err))
		}

		retryOpts = []retry.Option{
			retry.Delay(config.ReqDelay),
			retry.Attempts(uint(config.ReqTimeout / config.ReqDelay)),
			retry.DelayType(retry.FixedDelay),
		}
	}

	return retryOpts
}
