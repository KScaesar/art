package Artifex

// Lifecycle define a management mechanism when obj creation and obj end.
type Lifecycle struct {
	spawnHandlers []func() error
	exitHandlers  []func() error
	notifyExit    chan struct{}
}

func (life *Lifecycle) AddSpawnHandler(spawnHandlers ...func() error) {
	life.spawnHandlers = append(life.spawnHandlers, spawnHandlers...)
}

func (life *Lifecycle) AddExitHandler(exitHandlers ...func() error) {
	life.exitHandlers = append(life.exitHandlers, exitHandlers...)
}

func (life *Lifecycle) NotifyExit() {
	if life.notifyExit == nil {
		return
	}

	select {
	case <-life.notifyExit:
		return
	default:
		close(life.notifyExit)
	}
}

func (life *Lifecycle) Execute() error {
	if life.notifyExit != nil {
		return nil
	}

	err := life.spawn()
	if err != nil {
		return err
	}

	life.notifyExit = make(chan struct{})

	go func() {
		<-life.notifyExit
		life.exit()
	}()

	return nil
}

func (life *Lifecycle) spawn() error {
	if len(life.spawnHandlers) == 0 {
		return nil
	}
	for _, enter := range life.spawnHandlers {
		err := enter()
		if err != nil {
			life.exit()
			return err
		}
	}
	return nil
}

func (life *Lifecycle) exit() {
	if len(life.exitHandlers) == 0 {
		return
	}
	for _, action := range life.exitHandlers {
		action()
	}
	return
}
