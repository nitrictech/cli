package cloud

import (
	"github.com/nitrictech/cli/pkgplus/cloud/apis"
	"github.com/nitrictech/cli/pkgplus/cloud/collections"
	"github.com/nitrictech/cli/pkgplus/cloud/gateway"
	"github.com/nitrictech/cli/pkgplus/cloud/http"
	"github.com/nitrictech/cli/pkgplus/cloud/resources"
	"github.com/nitrictech/cli/pkgplus/cloud/schedules"
	"github.com/nitrictech/cli/pkgplus/cloud/secrets"
	"github.com/nitrictech/cli/pkgplus/cloud/storage"
	"github.com/nitrictech/cli/pkgplus/cloud/topics"
	"github.com/nitrictech/cli/pkgplus/cloud/websockets"
	"github.com/nitrictech/nitric/core/pkg/membrane"
	schedulespb "github.com/nitrictech/nitric/core/pkg/proto/schedules/v1"
	websocketspb "github.com/nitrictech/nitric/core/pkg/proto/websockets/v1"
	"github.com/samber/lo"
)

type Subscribable[T any] interface {
	SubscribeToState(fn func(T))
}
type LocalCloud struct {
	membrane *membrane.Membrane

	Apis        Subscribable[apis.State]
	Collections *collections.BoltDocService
	Gateway     *gateway.LocalGatewayService
	Http        *http.LocalHttpProxy
	Resources   *resources.LocalResourcesService
	Schedules   Subscribable[map[string]*schedulespb.RegistrationRequest]
	Secrets     *secrets.DevSecretService
	Storage     *storage.LocalStorageService
	Topics      Subscribable[map[string]int]
	Websockets  Subscribable[map[string][]websocketspb.WebsocketEventType]
}

// StartLocalNitric - starts the Nitric Server (membrane), including plugins and their local dependencies (e.g. local versions of cloud services
func (lc *LocalCloud) Stop() {
	lc.membrane.Stop()
	lc.Gateway.Stop()
}

// StartLocalNitric - starts the Nitric Server (membrane), including plugins and their local dependencies (e.g. local versions of cloud services
func (lc *LocalCloud) Start() error {
	return lc.membrane.Start()
}

func New() (*LocalCloud, error) {
	localTopics, err := topics.NewLocalTopicsService()
	if err != nil {
		return nil, err
	}

	localWebsockets, err := websockets.NewLocalWebsocketService()
	if err != nil {
		return nil, err
	}

	localStorage, err := storage.NewLocalStorageService(storage.StorageOptions{
		AccessKey: "dummykey",
		SecretKey: "dummysecret",
	})
	if err != nil {
		return nil, err
	}

	localApis := apis.NewLocalApiGatewayService()

	localSchedules := schedules.NewLocalSchedulesService()
	localHttpProxy := http.NewLocalHttpProxyService()

	localSecrets, err := secrets.NewSecretService()
	if err != nil {
		return nil, err
	}

	localGateway, err := gateway.NewGateway(localWebsockets)
	if err != nil {
		return nil, err
	}

	localResources := resources.NewLocalResourcesService(resources.LocalResourcesOptions{
		Gateway: localGateway,
	})

	collections, err := collections.NewBoltService()
	if err != nil {
		return nil, err
	}

	nitricMembraneServer, err := membrane.New(&membrane.MembraneOptions{
		// worker/listener plugins (these delegate incoming events/requests to handlers written with nitric)
		ApiPlugin:               localApis,
		HttpPlugin:              localHttpProxy,
		SchedulesPlugin:         localSchedules,
		TopicsListenerPlugin:    localTopics,
		StorageListenerPlugin:   localStorage,
		WebsocketListenerPlugin: localWebsockets,

		// address used by nitric clients to connect to the membrane (e.g. SDKs)
		ServiceAddress: "0.0.0.0:50051",

		// service plugins (these acloud services)
		SecretManagerPlugin: localSecrets,
		StoragePlugin:       localStorage,
		DocumentPlugin:      collections,
		GatewayPlugin:       localGateway,
		TopicsPlugin:        localTopics,
		ResourcesPlugin:     localResources,
		WebsocketPlugin:     localWebsockets,

		MinWorkers: lo.ToPtr(0),

		SuppressLogs: false,
	})
	if err != nil {
		return nil, err
	}

	return &LocalCloud{
		membrane:    nitricMembraneServer,
		Apis:        localApis,
		Http:        localHttpProxy,
		Resources:   localResources,
		Schedules:   localSchedules,
		Storage:     localStorage,
		Topics:      localTopics,
		Websockets:  localWebsockets,
		Gateway:     localGateway,
		Secrets:     localSecrets,
		Collections: collections,
	}, nil
}
