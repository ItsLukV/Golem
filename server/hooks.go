package server

func (ms *MinecraftServer) OnJoin(fn func(*Player) error) {
	ms.hmu.Lock()
	defer ms.hmu.Unlock()
	ms.onJoin = append(ms.onJoin, fn)
}

func (ms *MinecraftServer) OnMove(fn func(p *Player, prev PlayerPos) error) {
	ms.hmu.Lock()
	defer ms.hmu.Unlock()
	ms.onMove = append(ms.onMove, fn)
}

func (ms *MinecraftServer) OnLeave(fn func(*Player)) {
	ms.hmu.Lock()
	defer ms.hmu.Unlock()
	ms.onLeave = append(ms.onLeave, fn)
}

func (ms *MinecraftServer) OnUseItem(fn func(*Player) error) {
	ms.hmu.Lock()
	defer ms.hmu.Unlock()
	ms.onUseItem = append(ms.onUseItem, fn)
}

func (ms *MinecraftServer) OnPlayerInput(fn func(*Player, []InputType) error) {
	ms.hmu.Lock()
	defer ms.hmu.Unlock()
	ms.onPlayerInput = append(ms.onPlayerInput, fn)
}

func (ms *MinecraftServer) fireJoin(p *Player) error {
	ms.hmu.RLock()
	defer ms.hmu.RUnlock()
	for _, fn := range ms.onJoin {
		if err := fn(p); err != nil {
			return err
		}
	}
	return nil
}

func (ms *MinecraftServer) fireMove(p *Player, prev PlayerPos) error {
	ms.hmu.RLock()
	defer ms.hmu.RUnlock()
	for _, fn := range ms.onMove {
		if err := fn(p, prev); err != nil {
			return err
		}
	}
	return nil
}

func (ms *MinecraftServer) fireLeave(p *Player) {
	ms.hmu.RLock()
	defer ms.hmu.RUnlock()
	for _, fn := range ms.onLeave {
		fn(p)
	}
}

func (ms *MinecraftServer) fireOnUseItem(p *Player) error {
	ms.hmu.RLock()
	defer ms.hmu.RUnlock()
	for _, fn := range ms.onUseItem {
		if err := fn(p); err != nil {
			return err
		}
	}
	return nil
}

func (ms *MinecraftServer) fireOnPlayerInput(p *Player, input []InputType) error {
	ms.hmu.RLock()
	defer ms.hmu.RUnlock()
	for _, fn := range ms.onPlayerInput {
		if err := fn(p, input); err != nil {
			return err
		}
	}
	return nil
}
