package game

// HandleTelegramScore rejects all unsigned deltas, including negative values.
// Deprecated: use the canonical Telegram session/reward evidence flow.
func HandleTelegramScore(gs *GameState, player string, score int) error { return ErrLegacyGameDisabled }
