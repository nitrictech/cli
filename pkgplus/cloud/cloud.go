package cloud

import (
	"fmt"
	"sync"

	"github.com/samber/lo"
	"google.golang.org/grpc"

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
	"github.com/nitrictech/cli/pkgplus/grpcx"
	"github.com/nitrictech/cli/pkgplus/netx"
	"github.com/nitrictech/nitric/core/pkg/membrane"
)

type Subscribable[T any, A any] interface {
	SubscribeToState(fn func(T))
	SubscribeToAction(fn func(A)) // used to subscribe to api calls, ws messages, topic deliveries etc
}
type LocalCloud struct {
	membraneLock sync.Mutex
	membranes    map[string]*membrane.Membrane

	Apis        *apis.LocalApiGatewayService
	Collections *collections.BoltDocService
	Gateway     *gateway.LocalGatewayService
	Http        *http.LocalHttpProxy
	Resources   *resources.LocalResourcesService
	Schedules   *schedules.LocalSchedulesService
	Secrets     *secrets.DevSecretService
	Storage     *storage.LocalStorageService
	Topics      *topics.LocalTopicsAndSubscribersService
	Websockets  *websockets.LocalWebsocketService

	// Store all the plugins locally
}

// StartLocalNitric - starts the Nitric Server (membrane), including plugins and their local dependencies (e.g. local versions of cloud services
func (lc *LocalCloud) Stop() {
	for _, m := range lc.membranes {
		m.Stop()
	}
	lc.Gateway.Stop()
}

// StartLocalNitric - starts the Nitric Server (membrane), including plugins and their local dependencies (e.g. local versions of cloud services
// func (lc *LocalCloud) Start() error {
// 	errs, _ := errgroup.WithContext(context.Background())

// 	for serviceName, m := range lc.membranes {

// 		localMembrane := m

// 		errs.Go(func() error {
// 			interceptor, streamInterceptor := grpcx.CreateServiceIdInterceptor(serviceName)

// 			srv := grpc.NewServer(
// 				grpc.UnaryInterceptor(interceptor),
// 				grpc.StreamInterceptor(streamInterceptor),
// 			)

// 			return localMembrane.Start(membrane.WithGrpcServer(srv))
// 		})
// 	}

// 	return errs.Wait()
// }

func (lc *LocalCloud) AddService(serviceName string) (int, error) {
	lc.membraneLock.Lock()
	defer lc.membraneLock.Unlock()
	if _, ok := lc.membranes[serviceName]; ok {
		return 0, fmt.Errorf("service %s already started", serviceName)
	}

	// get an available port
	ports, err := netx.TakePort(1)
	if err != nil {
		return 0, err
	}

	nitricMembraneServer, err := membrane.New(&membrane.MembraneOptions{
		// worker/listener plugins (these delegate incoming events/requests to handlers written with nitric)
		ApiPlugin:               lc.Apis,
		HttpPlugin:              lc.Http,
		SchedulesPlugin:         lc.Schedules,
		TopicsListenerPlugin:    lc.Topics,
		StorageListenerPlugin:   lc.Storage,
		WebsocketListenerPlugin: lc.Websockets,

		// address used by nitric clients to connect to the membrane (e.g. SDKs)
		ServiceAddress: fmt.Sprintf("0.0.0.0:%d", ports[0]),

		// cloud service plugins
		SecretManagerPlugin: lc.Secrets,
		StoragePlugin:       lc.Storage,
		DocumentPlugin:      lc.Collections,
		GatewayPlugin:       lc.Gateway,
		TopicsPlugin:        lc.Topics,
		ResourcesPlugin:     lc.Resources,
		WebsocketPlugin:     lc.Websockets,

		MinWorkers: lo.ToPtr(0),

		SuppressLogs: false,
	})

	// Create a watcher that clears old resources when the service is restarted
	_, err = resources.NewServiceResourceRefresher(serviceName, resources.NewServiceResourceRefresherArgs{
		Resources:  lc.Resources,
		Apis:       lc.Apis,
		Schedules:  lc.Schedules,
		Http:       lc.Http,
		Listeners:  lc.Storage,
		Websockets: lc.Websockets,
		Topics:     lc.Topics,
		Storage:    lc.Storage,
	})
	if err != nil {
		return 0, err
	}

	go func() {
		interceptor, streamInterceptor := grpcx.CreateServiceNameInterceptor(serviceName)

		srv := grpc.NewServer(
			grpc.UnaryInterceptor(interceptor),
			grpc.StreamInterceptor(streamInterceptor),
		)

		nitricMembraneServer.Start(membrane.WithGrpcServer(srv))
	}()

	lc.membranes[serviceName] = nitricMembraneServer
	return ports[0], nil
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

	localGateway, err := gateway.NewGateway()
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

	return &LocalCloud{
		membranes:   make(map[string]*membrane.Membrane),
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
