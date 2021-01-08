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

			sendResponse := func(request ipamspec.IPAMRequest, ipAddr string) {
				resp := ipamspec.IPAMResponse{
					Request: request,
					IPAddr:  ipAddr,
					Status:  true,
				}
				ctlr.respChan <- resp
			}

			ipAddr := ctlr.Manager.GetIPAddress(req.HostName)
			if ipAddr != "" {
				go sendResponse(req, ipAddr)
				break
			}

			ipAddr = ctlr.Manager.GetNextIPAddress(req.CIDR)
			if ipAddr != "" {
				log.Debugf("Allocated IP: %v for CIDR: %v", ipAddr, req.CIDR)
				ctlr.Manager.CreateARecord(req.HostName, ipAddr)
				go sendResponse(req, ipAddr)
			}
		case ipamspec.DELETE:
			ipAddr := ctlr.Manager.GetIPAddress(req.HostName)
			if ipAddr != "" {
				ctlr.Manager.ReleaseIPAddress(ipAddr)
				ctlr.Manager.DeleteARecord(req.CIDR, ipAddr)
			}
			go func(request ipamspec.IPAMRequest) {
				resp := ipamspec.IPAMResponse{
					Request: request,
					IPAddr:  "",
					Status:  true,
				}
				ctlr.respChan <- resp
			}(req)
		}
	}
}

func (ctlr *Controller) Start() {
	ctlr.Orchestrator.SetupCommunicationChannels(
		ctlr.reqChan,
		ctlr.respChan,
	)
	log.Info("Controller started")

	ctlr.Orchestrator.Start(ctlr.StopCh)

	go ctlr.runController()
}

func (ctlr *Controller) Stop() {
	ctlr.Orchestrator.Stop()
}
