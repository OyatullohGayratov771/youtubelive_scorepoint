package state

import "sync"

type GameState struct {
	Player1Name string `json:"player1_name"`
	Player2Name string `json:"player2_name"`

	// Joriy ramka ichidagi tosh hisobi (har ramka boshida sifirlanadi)
	Player1FrameScore int `json:"player1_frame_score"`
	Player2FrameScore int `json:"player2_frame_score"`

	// Match hisobi: nechta ramka yutildi
	Player1Games int `json:"player1_games"`
	Player2Games int `json:"player2_games"`

	// Nechta ramka yutsa match g'olib bo'ladi
	RaceTo int `json:"race_to"`

	// Match g'olibi (bo'sh = hali aniqlanmagan)
	Winner string `json:"winner"`
}

type Manager struct {
	mu       sync.RWMutex
	state    GameState
	onChange func(GameState)
}

func New(onChange func(GameState)) *Manager {
	return &Manager{
		state: GameState{
			Player1Name: "Player 1",
			Player2Name: "Player 2",
			RaceTo:      7,
		},
		onChange: onChange,
	}
}

func (m *Manager) Get() GameState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state
}

func (m *Manager) Update(fn func(*GameState)) {
	m.mu.Lock()
	fn(&m.state)
	s := &m.state
	// Match g'olibini avtomatik aniqlash
	if s.Winner == "" {
		if s.Player1Games >= s.RaceTo {
			s.Winner = s.Player1Name
		} else if s.Player2Games >= s.RaceTo {
			s.Winner = s.Player2Name
		}
	}
	gs := m.state
	m.mu.Unlock()
	if m.onChange != nil {
		m.onChange(gs)
	}
}

// P1WinsFrame — P1 ramkani yutdi: +1 game, frame sifirlanadi
func (m *Manager) P1WinsFrame() {
	m.Update(func(s *GameState) {
		if s.Winner != "" {
			return
		}
		s.Player1Games++
		s.Player1FrameScore = 0
		s.Player2FrameScore = 0
	})
}

// P2WinsFrame — P2 ramkani yutdi: +1 game, frame sifirlanadi
func (m *Manager) P2WinsFrame() {
	m.Update(func(s *GameState) {
		if s.Winner != "" {
			return
		}
		s.Player2Games++
		s.Player1FrameScore = 0
		s.Player2FrameScore = 0
	})
}

// ResetFrame — faqat joriy frame scoreni sifirla
func (m *Manager) ResetFrame() {
	m.Update(func(s *GameState) {
		s.Player1FrameScore = 0
		s.Player2FrameScore = 0
	})
}

// NewMatch — to'liq yangi match
func (m *Manager) NewMatch() {
	m.Update(func(s *GameState) {
		s.Player1Games = 0
		s.Player2Games = 0
		s.Player1FrameScore = 0
		s.Player2FrameScore = 0
		s.Winner = ""
	})
}
