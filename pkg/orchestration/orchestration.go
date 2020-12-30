package orchestration

import (
	"github.com/subbuv26/f5-ipam-controller/pkg/ipamspec"
)

type Orchestrator interface {
	// SetupCommunicationChannels sets Request and Response channels
	SetupCommunicationChannels(reqChan chan<- ipamspec.IPAMRequest, respChan <-chan ipamspec.IPAMResponse)
	// Runs the Orchestrator, watching for resources
	Run(stopCh <-chan struct{})
}

func NewOrchestrator() Orchestrator {
	return nil
}
