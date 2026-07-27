//go:build !libvips

package imagevips

type Runtime struct{}

func NewRuntime() *Runtime {
	return &Runtime{}
}

func (*Runtime) Start() error {
	return nil
}

func (*Runtime) Shutdown() {}
