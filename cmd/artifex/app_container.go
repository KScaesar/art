package leaf

import (
	"code.xasxly.com/platform-server/artemis.git/pkg/pubsub"
)

type Application struct {
	Websocket
}

func (app *Application) Init(ingressMux *IngressMux, egressMux *EgressMux) {

}

//

type ApplicationContainer interface {
	NextApp()
	IngressMux() *IngressMux
	EgressMux(*GorillaConn) *EgressMux
	DecorateAdapter(adp pubsub.IAdapter) (app pubsub.IAdapter)
	Lifecycle(lifecycle *pubsub.Lifecycle)
}

func NewAppContainer(newApp func() *Application) ApplicationContainer {
	app := newApp()
	return &AppContainer{
		newApp:     newApp,
		app:        app,
		egressMux:  NewEgressMux(),
		ingressMux: NewIngressMux(),
	}
}

type AppContainer struct {
	newApp func() *Application
	app    *Application

	egressMux  *EgressMux
	ingressMux *IngressMux
}

func (container *AppContainer) NextApp() {
	container.app = container.newApp()
	container.egressMux = NewEgressMux()
}

func (container *AppContainer) IngressMux() *IngressMux {
	mux := container.ingressMux

	return mux
}

func (container *AppContainer) EgressMux(conn *GorillaConn) *EgressMux {
	mux := container.egressMux

	return mux
}

func (container *AppContainer) DecorateAdapter(adp pubsub.IAdapter) pubsub.IAdapter {
	app := container.app
	app.Websocket = adp.(Websocket)
	return app
}

func (container *AppContainer) Lifecycle(lifecycle *pubsub.Lifecycle) {
	lifecycle.OnOpen(func(adp pubsub.IAdapter) error {
		container.app.Init(container.ingressMux, container.egressMux)
		return nil
	})
}

//

type SimpleAppContainer struct {
	IngressMux_f      func() *IngressMux
	EgressMux_f       func(*GorillaConn) *EgressMux
	DecorateAdapter_f func(adp pubsub.IAdapter) (app pubsub.IAdapter)
	Lifecycle_f       func(lifecycle *pubsub.Lifecycle)
}

func (SimpleAppContainer) NextApp() {}

func (container *SimpleAppContainer) IngressMux() *IngressMux {
	return container.IngressMux_f()
}

func (container *SimpleAppContainer) EgressMux(conn *GorillaConn) *EgressMux {
	return container.EgressMux_f(conn)
}

func (container *SimpleAppContainer) DecorateAdapter(adp pubsub.IAdapter) (app pubsub.IAdapter) {
	return container.DecorateAdapter_f(adp)
}

func (container *SimpleAppContainer) Lifecycle(lifecycle *pubsub.Lifecycle) {
	if container.Lifecycle_f != nil {
		container.Lifecycle_f(lifecycle)
	}
}
