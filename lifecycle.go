package Artifex

import (
	"sync"
)

// Lifecycle define a management mechanism when obj creation and obj end.
type Lifecycle struct {
	installHandlers   []func() error
	uninstallHandlers []func()
	wg                sync.WaitGroup
}

func (life *Lifecycle) AddInstall(installs ...func() error) {
	life.installHandlers = append(life.installHandlers, installs...)
}

func (life *Lifecycle) AddUninstall(uninstalls ...func()) {
	life.uninstallHandlers = append(life.uninstallHandlers, uninstalls...)
}

func (life *Lifecycle) Install() error {
	for _, install := range life.installHandlers {
		err := install()
		if err != nil {
			life.Uninstall()
			return err
		}
	}
	return nil
}

func (life *Lifecycle) Uninstall() {
	for _, uninstall := range life.uninstallHandlers {
		uninstall()
	}
	return
}

func (life *Lifecycle) AsyncUninstall() {
	for _, h := range life.uninstallHandlers {
		uninstall := h
		life.wg.Add(1)
		go func() {
			defer life.wg.Done()
			uninstall()
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
