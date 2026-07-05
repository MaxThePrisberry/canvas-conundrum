package game

// Resource-gathering phase. The round scheduler, question delivery, and
// token economy are implemented in milestone M4; this file currently holds
// the entry point and reconnect-replay hooks the engine core calls.

// enterResourceGathering transitions setup → resource_gathering.
func (e *Engine) enterResourceGathering() {
	e.phase = "resource_gathering"
}

// handleResourceTimer dispatches the phase's named timers.
func (e *Engine) handleResourceTimer(name string) {}

// replayHostResourceState restores host state after a mid-phase reconnect.
func (e *Engine) replayHostResourceState() {}

// replayPlayerResourceState restores player state after a mid-phase reconnect.
func (e *Engine) replayPlayerResourceState(p *Player) {}
