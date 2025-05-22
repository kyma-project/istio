package testcontext

import (
	"fmt"
	"time"

	"github.com/avast/retry-go"
	"github.com/vrischmann/envconfig"
)

type Config struct {
	ClientTimeout time.Duration `envconfig:"TEST_CLIENT_TIMEOUT,default=10s"`
	ReqTimeout    time.Duration `envconfig:"TEST_REQUEST_TIMEOUT,default=300s"`
	ReqDelay      time.Duration `envconfig:"TEST_REQUEST_DELAY,default=5s"`
}

var (
	retryOpts []retry.Option
)

func GetRetryOpts() []retry.Option {
	if retryOpts == nil {
		var config Config
		if err := envconfig.Init(&config); err != nil {
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
