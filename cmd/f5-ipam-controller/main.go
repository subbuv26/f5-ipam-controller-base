package main

import (
	"fmt"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"os/signal"
	"strings"
	"syscall"

	flag "github.com/spf13/pflag"
	"github.com/subbuv26/f5-ipam-controller/pkg/controller"
	"github.com/subbuv26/f5-ipam-controller/pkg/manager"
	"github.com/subbuv26/f5-ipam-controller/pkg/orchestration"
	log "github.com/subbuv26/f5-ipam-controller/pkg/vlogger"
	clog "github.com/subbuv26/f5-ipam-controller/pkg/vlogger/console"
)

const (
	DefaultProvider = "f5-ip-provider"
)

var (
	// Flag sets and supported flags
	flags         *flag.FlagSet
	globalFlags   *flag.FlagSet
	kubeFlags     *flag.FlagSet
	providerFlags *flag.FlagSet

	// Global
	logLevel *string
	orch     *string
	provider *string

	// Kubernetes
	inCluster  *bool
	kubeConfig *string

	// Provider
	iprange *string
)

func init() {
	flags = flag.NewFlagSet("main", flag.ContinueOnError)
	globalFlags = flag.NewFlagSet("Global", flag.ContinueOnError)
	kubeFlags = flag.NewFlagSet("Kubernetes", flag.ContinueOnError)
	providerFlags = flag.NewFlagSet("Provider", flag.ContinueOnError)

	//Flag terminal wrapping
	var err error
	var width int
	fd := int(os.Stdout.Fd())
	if terminal.IsTerminal(fd) {
		width, _, err = terminal.GetSize(fd)
		if nil != err {
			width = 0
		}
	}

	// Global flags
	logLevel = globalFlags.String("log-level", "INFO", "Optional, logging level.")
	orch = globalFlags.String("orchestration", "",
		"Required, orchestration that the controller is running in.")
	provider = globalFlags.String("ip-provider", DefaultProvider,
		"Required, the IPAM system that the controller will interface with.")

	// Kubernetes flags
	inCluster = kubeFlags.Bool("running-in-cluster", true,
		"Optional, if this controller is running in a Kubernetes cluster, "+
			"use the pod secrets for creating a Kubernetes client.")
	kubeConfig = kubeFlags.String("kubeconfig", "./config",
		"Optional, absolute path to the kubeconfig file.")

	iprange = providerFlags.String("iprange", "",
		"Optional, the Default Provider needs iprange to build pools of IP Addresses")

	globalFlags.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "  Global:\n%s\n", globalFlags.FlagUsagesWrapped(width))
	}

	kubeFlags.Usage = func() {
		fmt.Fprintf(os.Stderr, "  Kubernetes:\n%s\n", kubeFlags.FlagUsagesWrapped(width))
	}

	providerFlags.Usage = func() {
		_, _ = fmt.Fprintf(os.Stderr, "  Provider:\n%s\n", providerFlags.FlagUsagesWrapped(width))
	}
	flags.AddFlagSet(globalFlags)
	flags.AddFlagSet(kubeFlags)
	flags.AddFlagSet(providerFlags)

	flags.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s\n", os.Args[0])
		globalFlags.Usage()
		kubeFlags.Usage()
		providerFlags.Usage()
	}
	_, _ = fmt.Fprintf(os.Stderr, "  Temp: %s", *iprange)
}

func verifyArgs() error {
	log.RegisterLogger(
		log.LL_MIN_LEVEL, log.LL_MAX_LEVEL, clog.NewConsoleLogger())

	if ll := log.NewLogLevel(*logLevel); nil != ll {
		log.SetLogLevel(*ll)
	} else {
		return fmt.Errorf("Unknown log level requested: %v\n"+
			"    Valid log levels are: DEBUG, INFO, WARNING, ERROR, CRITICAL", logLevel)
	}

	if len(*orch) == 0 {
		return fmt.Errorf("orchestration is required")
	}

	*orch = strings.ToLower(*orch)
	*provider = strings.ToLower(*provider)

	return nil
}

func main() {
	err := flags.Parse(os.Args)
	if nil != err {
		os.Exit(1)
	}

	err = verifyArgs()
	if nil != err {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		flags.Usage()
		os.Exit(1)
	}

	orcr := orchestration.NewOrchestrator()
	mgrParams := manager.Params{
		Provider:          *provider,
		IPAMManagerParams: manager.IPAMManagerParams{Range: *iprange},
	}
	mgrParams.Range = *iprange
	mgr := manager.NewManager(mgrParams)
	stopCh := make(chan struct{})

	ctlr := controller.NewController(
		controller.Spec{
			Orchestrator: orcr,
			Manager:      mgr,
			StopCh:       stopCh,
		},
	)
	ctlr.Run()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	sig := <-signals

	ctlr.Stop()
	log.Infof("Exiting - signal %v\n", sig)
	close(stopCh)
}
