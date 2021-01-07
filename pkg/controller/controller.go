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

			sendResponse := func(ipAddr string) {
				resp := ipamspec.IPAMResponse{
					Request: req,
					IPAddr:  ipAddr,
					Status:  true,
				}
				ctlr.respChan <- resp
			}

			ipAddr := ctlr.Manager.GetIPAddress(req.HostName)
			if ipAddr != "" {
				go sendResponse(ipAddr)
				continue
			}

			ipAddr = ctlr.Manager.GetNextIPAddress(req.CIDR)
			if ipAddr != "" {
				log.Debugf("Allocated IP: %v for CIDR: %v", ipAddr, req.CIDR)
				ctlr.Manager.CreateARecord(req.HostName, ipAddr)
				go sendResponse(ipAddr)
			}
		case ipamspec.DELETE:
			ctlr.Manager.ReleaseIPAddress(req.IPAddr)
			ctlr.Manager.DeleteARecord(req.CIDR, req.IPAddr)
			go func() {
				resp := ipamspec.IPAMResponse{
					Request: req,
					IPAddr:  "",
					Status:  true,
				}
				ctlr.respChan <- resp
			}()
		}
	}
}

func (ctlr *Controller) Start() {
	ctlr.Orchestrator.SetupCommunicationChannels(
		ctlr.reqChan,
		ctlr.respChan,
	)
	log.Infof("Controller started: (%p)", ctlr)

	ctlr.Orchestrator.Start(ctlr.StopCh)

	go ctlr.runController()
}

func (ctlr *Controller) Stop() {
	ctlr.Orchestrator.Stop()
}
