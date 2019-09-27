package main

import (
	"context"
	"strings"

	fakeProvider "fake-vn/provider"

	"github.com/sirupsen/logrus"

	cli "github.com/virtual-kubelet/node-cli"
	logruscli "github.com/virtual-kubelet/node-cli/logrus"
	"github.com/virtual-kubelet/node-cli/opts"
	"github.com/virtual-kubelet/node-cli/provider"
	"github.com/virtual-kubelet/virtual-kubelet/log"
	logruslogger "github.com/virtual-kubelet/virtual-kubelet/log/logrus"
)

var (
	buildVersion = "N/A"
	buildTime    = "N/A"
)

func main() {
	ctx := cli.ContextWithCancelOnSignal(context.Background())

	logger := logrus.StandardLogger()
	log.L = logruslogger.FromLogrus(logrus.NewEntry(logger))
	logConfig := &logruscli.Config{LogLevel: "info"}

	op := opts.New()
	op.Provider = "fake-vn"
	op.Version = strings.Join([]string{"1.15.0", "fake", "0.0.0"}, "-")
	op.DisableTaint = true

	node, err := cli.New(ctx,
		cli.WithBaseOpts(op),
		cli.WithCLIVersion(buildVersion, buildTime),
		cli.WithKubernetesNodeVersion("v1.14.3"),
		cli.WithProvider("fake-vn", func(cfg provider.InitConfig) (provider.Provider, error) {
			return fakeProvider.NewNoOpProvider(
				cfg.ResourceManager,
				cfg.NodeName,
				cfg.OperatingSystem,
				cfg.InternalIP,
				cfg.DaemonPort)
		}),
		nil,
		cli.WithPersistentPreRunCallback(func() error {
			return logruscli.Configure(logConfig, logger)
		}),
	)

	if err != nil {
		log.G(ctx).Fatal(err)
	}

	if err := node.Run(); err != nil {
		log.G(ctx).Fatal(err)
	}
}
