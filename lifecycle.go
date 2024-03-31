package Artifex

// Lifecycle define a management mechanism when obj creation and obj end.
type Lifecycle struct {
	installHandlers   []func() error
	uninstallHandlers []func()
}

func (life *Lifecycle) AddInstall(installs ...func() error) {
	life.installHandlers = append(life.installHandlers, installs...)
}

func (life *Lifecycle) AddUninstall(uninstalls ...func()) {
	life.uninstallHandlers = append(life.uninstallHandlers, uninstalls...)
}

func (life *Lifecycle) DoInstall() error {
	for _, install := range life.installHandlers {
		err := install()
		if err != nil {
			life.DoUninstall()
			return err
		}
	}
	return nil
}

func (life *Lifecycle) DoUninstall() {
	for _, uninstall := range life.uninstallHandlers {
		uninstall()
	}
	return
}

func (life *Lifecycle) InstallQty() int {
	return len(life.installHandlers)
}

func (life *Lifecycle) UninstallQty() int {
	return len(life.uninstallHandlers)
}
