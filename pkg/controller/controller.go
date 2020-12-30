package controller

import (
	"github.com/subbuv26/f5-ipam-controller/pkg/ipamspec"
	"github.com/subbuv26/f5-ipam-controller/pkg/manager"
	"github.com/subbuv26/f5-ipam-controller/pkg/orchestration"
	log "github.com/subbuv26/f5-ipam-controller/pkg/vlogger"
)

type Spec struct {
	Orchestrator orchestration.Orchestrator
	Manager      manager.Manager
	StopCh       chan struct{}
}

type Controller struct {
	Spec
	reqChan  chan ipamspec.IPAMRequest
	respChan chan ipamspec.IPAMResponse
}

func NewController(spec Spec) *Controller {
	ctlr := &Controller{
		Spec:     spec,
		reqChan:  make(chan ipamspec.IPAMRequest),
		respChan: make(chan ipamspec.IPAMResponse),
	}

	return ctlr
}

func (ctlr *Controller) runController() {
	for req := range ctlr.reqChan {
		switch req.Operation {
		case ipamspec.CREATE:
			ipAddr := ctlr.Manager.GetNextAddr(req.CIDR)
			if ipAddr != "" {
				ctlr.Manager.CreateARecord(req.HostName, ipAddr)
				go func() {
					resp := ipamspec.IPAMResponse{
						Request: req,
						IPAddr:  ipAddr,
					}
					ctlr.respChan <- resp
				}()
			}
			//case ipamspec.DELETE:
			//	ctlr.Manager.ReleaseAddr(req.CIDR)
		}
	}
}

func (ctlr *Controller) Run() {
	ctlr.Orchestrator.SetupCommunicationChannels(
		ctlr.reqChan,
		ctlr.respChan,
	)
	log.Infof("Controller started: (%p)", ctlr)

	ctlr.Orchestrator.Run(ctlr.StopCh)

	go ctlr.runController()
}
