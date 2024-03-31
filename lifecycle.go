package Artifex

import (
	"sync"
)

// Lifecycle define a management mechanism when obj creation and obj end.
type Lifecycle struct {
	installHandlers   []func(adapter IAdapter) error
	uninstallHandlers []func(adapter IAdapter)
	wg                sync.WaitGroup
}

func (life *Lifecycle) AddInstall(installs ...func(adp IAdapter) error) *Lifecycle {
	life.installHandlers = append(life.installHandlers, installs...)
	return life
}

func (life *Lifecycle) AddUninstall(uninstalls ...func(adp IAdapter)) *Lifecycle {
	life.uninstallHandlers = append(life.uninstallHandlers, uninstalls...)
	return life
}

func (life *Lifecycle) Install(adp IAdapter) error {
	for _, install := range life.installHandlers {
		err := install(adp)
		if err != nil {
			life.Uninstall(adp)
			return err
		}
	}
	return nil
}

func (life *Lifecycle) Uninstall(adp IAdapter) {
	for _, uninstall := range life.uninstallHandlers {
		uninstall(adp)
	}
	return
}

func (life *Lifecycle) AsyncUninstall(adp IAdapter) {
	for _, h := range life.uninstallHandlers {
		uninstall := h
		life.wg.Add(1)
		go func() {
			defer life.wg.Done()
			uninstall(adp)
		}()
	}
	return
}

func (life *Lifecycle) Wait() {
	life.wg.Wait()
}

func (life *Lifecycle) InstallQty() int {
	return len(life.installHandlers)
}

func (life *Lifecycle) UninstallQty() int {
	return len(life.uninstallHandlers)
}
