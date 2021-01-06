package orchestration

import (
	ficV1 "github.com/subbuv26/f5-ipam-controller/pkg/ipamapis/apis/fic/v1"
	"github.com/subbuv26/f5-ipam-controller/pkg/ipammachinery"
	"github.com/subbuv26/f5-ipam-controller/pkg/ipamspec"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"time"

	log "github.com/subbuv26/f5-ipam-controller/pkg/vlogger"
	//
	//"k8s.io/api/core/v1"
	//"k8s.io/api/extensions/v1beta1"
	//"k8s.io/apimachinery/pkg/api/meta"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//"k8s.io/apimachinery/pkg/labels"
	//"k8s.io/apimachinery/pkg/runtime"
	//utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"

	//"k8s.io/client-go/rest"
	//"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type K8sIPAMClient struct {
	ipamCli *ipammachinery.IPAMClient

	// Queue and informers for namespaces and resources
	rscQueue workqueue.RateLimitingInterface

	// Channel for sending request to controller
	reqChan chan<- ipamspec.IPAMRequest
	// Channel for receiving responce from controller
	respChan <-chan ipamspec.IPAMResponse
}

const (
	CREATE = "Create"
	UPDATE = "Update"
	DELETE = "Delete"

	DefaultNamespace = "kube-system"
)

type rqKey struct {
	rsc       *ficV1.F5IPAM
	oldRsc    *ficV1.F5IPAM
	Operation string
}

type specMap map[ficV1.HostSpec]bool

type ResourceMeta struct {
	name      string
	namespace string
}

func NewIPAMK8SClient() *K8sIPAMClient {
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Fatalf("Error creating configuration: %v", err)
		return nil
	}
	k8sIPAMClient := &K8sIPAMClient{
		rscQueue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(), "ipam-controller"),
	}

	eventHandlers := &cache.ResourceEventHandlerFuncs{
		AddFunc:    func(obj interface{}) { k8sIPAMClient.enqueueIPAM(obj) },
		UpdateFunc: func(oldObj, newObj interface{}) { k8sIPAMClient.enqueueUpdatedIPAM(oldObj, newObj) },
		DeleteFunc: func(obj interface{}) { k8sIPAMClient.enqueueDeletedIPAM(obj) },
	}

	ipamParams := ipammachinery.Params{
		Config:        config,
		EventHandlers: eventHandlers,
		Namespaces:    []string{DefaultNamespace},
	}

	ipamCli := ipammachinery.NewIPAMClient(ipamParams)

	k8sIPAMClient.ipamCli = ipamCli
	return k8sIPAMClient
}

// SetupCommunicationChannels sets Request and Response channels
func (k8sc *K8sIPAMClient) SetupCommunicationChannels(
	reqChan chan<- ipamspec.IPAMRequest,
	respChan <-chan ipamspec.IPAMResponse,
) {
	k8sc.reqChan = reqChan
	k8sc.respChan = respChan
}

// Runs the Orchestrator, watching for resources
func (k8sc *K8sIPAMClient) Run(stopCh <-chan struct{}) {
	k8sc.ipamCli.Start()
	go wait.Until(k8sc.customResourceWorker, time.Second, stopCh)
	go wait.Until(k8sc.responseWorker, time.Second, stopCh)
}

func (k8sc *K8sIPAMClient) Stop() {
	k8sc.ipamCli.Stop()
}

func (k8sc *K8sIPAMClient) enqueueIPAM(obj interface{}) {
	key := rqKey{
		rsc:       obj.(*ficV1.F5IPAM),
		oldRsc:    nil,
		Operation: CREATE,
	}

	k8sc.rscQueue.Add(key)
}

func (k8sc *K8sIPAMClient) enqueueUpdatedIPAM(old, cur interface{}) {
	key := rqKey{
		rsc:       cur.(*ficV1.F5IPAM),
		oldRsc:    old.(*ficV1.F5IPAM),
		Operation: UPDATE,
	}

	k8sc.rscQueue.Add(key)
}

func (k8sc *K8sIPAMClient) enqueueDeletedIPAM(obj interface{}) {
	key := rqKey{
		rsc:       obj.(*ficV1.F5IPAM),
		oldRsc:    nil,
		Operation: DELETE,
	}

	k8sc.rscQueue.Add(key)
}

// customResourceWorker starts the Custom Resource Worker.
func (k8sc *K8sIPAMClient) customResourceWorker() {
	log.Debugf("Starting Custom Resource Worker")
	for k8sc.processResource() {
	}
}

func (k8sc *K8sIPAMClient) responseWorker() {
	log.Debugf("Starting Response Worker")
	for k8sc.processResponse() {
	}
}

func (k8sc *K8sIPAMClient) processResource() bool {
	key, quit := k8sc.rscQueue.Get()
	if quit {
		// The controller is shutting down.
		log.Debugf("Resource Queue is empty, Going to StandBy Mode")
		return false
	}

	defer k8sc.rscQueue.Done(key)
	rKey := key.(*rqKey)
	log.Debugf("Processing Key: %v", rKey)

	switch rKey.Operation {
	case CREATE:
		// A new CIS has created a new IPAM CR or FIC restarted

		// On FIC restart build already allocated IPSpec Store from the Status of F5IPAM CR
		for _, ipSpec := range rKey.rsc.Status.IPStatus {
			ipamReq := ipamspec.IPAMRequest{
				Metadata: ResourceMeta{
					name:      rKey.rsc.Name,
					namespace: rKey.rsc.Namespace,
				},
				HostName:  ipSpec.Host,
				CIDR:      ipSpec.Cidr,
				IPAddr:    ipSpec.IP,
				Operation: ipamspec.CREATE,
			}
			k8sc.reqChan <- ipamReq
		}

		for _, hostSpec := range rKey.rsc.Spec.HostSpecs {
			ipamReq := ipamspec.IPAMRequest{
				Metadata: ResourceMeta{
					name:      rKey.rsc.Name,
					namespace: rKey.rsc.Namespace,
				},
				HostName:  hostSpec.Host,
				CIDR:      hostSpec.Cidr,
				Operation: ipamspec.CREATE,
			}
			k8sc.reqChan <- ipamReq
		}
	case DELETE:
		for _, ipStatus := range rKey.rsc.Status.IPStatus {
			ipamReq := ipamspec.IPAMRequest{
				Metadata: ResourceMeta{
					name:      rKey.rsc.Name,
					namespace: rKey.rsc.Namespace,
				},
				HostName:  ipStatus.Host,
				CIDR:      ipStatus.Cidr,
				IPAddr:    ipStatus.IP,
				Operation: ipamspec.DELETE,
			}
			k8sc.reqChan <- ipamReq
		}
	case UPDATE:
		oldSpecSet := make(specMap)
		newSpecSet := make(specMap)
		for _, hostSpec := range rKey.oldRsc.Spec.HostSpecs {
			oldSpecSet[*hostSpec] = true
		}
		for _, hostSpec := range rKey.rsc.Spec.HostSpecs {
			newSpecSet[*hostSpec] = true
		}

		for spec, _ := range oldSpecSet {
			if _, ok := newSpecSet[spec]; !ok {
				// This spec got deleted
				ipamReq := ipamspec.IPAMRequest{
					Metadata: ResourceMeta{
						name:      rKey.rsc.Name,
						namespace: rKey.rsc.Namespace,
					},
					HostName:  spec.Host,
					CIDR:      spec.Cidr,
					Operation: ipamspec.DELETE,
				}
				k8sc.reqChan <- ipamReq
			}
		}

		for spec, _ := range newSpecSet {
			if _, ok := oldSpecSet[spec]; !ok {
				ipamReq := ipamspec.IPAMRequest{
					Metadata: ResourceMeta{
						name:      rKey.rsc.Name,
						namespace: rKey.rsc.Namespace,
					},
					HostName:  spec.Host,
					CIDR:      spec.Cidr,
					Operation: ipamspec.CREATE,
				}
				k8sc.reqChan <- ipamReq
			}
		}

	}
	return true
}

func (k8sc *K8sIPAMClient) processResponse() bool {
	for resp := range k8sc.respChan {
		switch resp.Request.Operation {
		case ipamspec.CREATE:
			if resp.Status {
				metadata := resp.Request.Metadata.(ResourceMeta)
				ipamRsc, err := k8sc.ipamCli.Get(metadata.namespace, metadata.name)
				if err != nil {
					log.Errorf("Unable to find F5IPAM: %v/%v to update", metadata.namespace, metadata.name)
				}

				var found bool
				for _, ipSpec := range ipamRsc.Status.IPStatus {
					if ipSpec.Host == resp.Request.HostName &&
						ipSpec.Cidr == resp.Request.CIDR {

						ipSpec.IP = resp.IPAddr
						found = true
					}
				}
				if !found {
					ipSpec := &ficV1.IPSpec{
						Host: resp.Request.HostName,
						Cidr: resp.Request.CIDR,
						IP:   resp.IPAddr,
					}
					ipamRsc.Status.IPStatus = append(ipamRsc.Status.IPStatus, ipSpec)
				}

				_, err = k8sc.ipamCli.Update(ipamRsc.Namespace, ipamRsc)
				if err != nil {
					log.Errorf("Unable to Update F5IPAM: %v/%v", metadata.namespace, metadata.name)
				}
			}
		}
	}
	return true
}
