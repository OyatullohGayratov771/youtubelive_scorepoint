package bot

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/oyatulloh/scorepoint/internal/state"
)

type session int

const (
	idle session = iota
	awaitP1Name
	awaitP2Name
	awaitRaceTo
)

type Bot struct {
	api      *tgbotapi.BotAPI
	sm       *state.Manager
	sessions map[int64]session
}

func New(token string, sm *state.Manager) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	log.Printf("bot: @%s tayyor", api.Self.UserName)
	return &Bot{api: api, sm: sm, sessions: make(map[int64]session)}, nil
}

func (b *Bot) Run() {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	for upd := range b.api.GetUpdatesChan(u) {
		if upd.CallbackQuery != nil {
			b.handleCallback(upd.CallbackQuery)
		} else if upd.Message != nil {
			b.handleMessage(upd.Message)
		}
	}
}

func (b *Bot) handleMessage(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	text := strings.TrimSpace(msg.Text)

	switch b.sessions[chatID] {
	case awaitP1Name:
		if text != "" {
			b.sm.Update(func(s *state.GameState) { s.Player1Name = text })
		}
		b.sessions[chatID] = idle
		b.sendPanel(chatID)
		return
	case awaitP2Name:
		if text != "" {
			b.sm.Update(func(s *state.GameState) { s.Player2Name = text })
		}
		b.sessions[chatID] = idle
		b.sendPanel(chatID)
		return
	case awaitRaceTo:
		if n, err := strconv.Atoi(text); err == nil && n > 0 && n <= 999 {
			b.sm.Update(func(s *state.GameState) { s.RaceTo = n; s.Winner = "" })
		}
		b.sessions[chatID] = idle
		b.sendPanel(chatID)
		return
	}
	b.sendPanel(chatID)
}

func (b *Bot) handleCallback(cb *tgbotapi.CallbackQuery) {
	chatID := cb.Message.Chat.ID
	msgID := cb.Message.MessageID
	b.api.Request(tgbotapi.NewCallback(cb.ID, ""))

	switch cb.Data {

	// ── FRAME SCORE (tosh hisobi) ──────────────
	case "p1_fp":
		b.sm.Update(func(s *state.GameState) {
			if s.Winner == "" {
				s.Player1FrameScore++
			}
		})
	case "p1_fm":
		b.sm.Update(func(s *state.GameState) {
			if s.Player1FrameScore > 0 {
				s.Player1FrameScore--
			}
		})
	case "p2_fp":
		b.sm.Update(func(s *state.GameState) {
			if s.Winner == "" {
				s.Player2FrameScore++
			}
		})
	case "p2_fm":
		b.sm.Update(func(s *state.GameState) {
			if s.Player2FrameScore > 0 {
				s.Player2FrameScore--
			}
		})

	// ── FRAME YUTDI (match game +1, frame sifirlanadi) ─
	case "p1_wins":
		b.sm.P1WinsFrame()
	case "p2_wins":
		b.sm.P2WinsFrame()

	// ── MATCH GAME TUZATISH (xato bo'lsa qaytarish) ──
	case "p1_gm":
		b.sm.Update(func(s *state.GameState) {
			if s.Player1Games > 0 {
				s.Player1Games--
				s.Winner = ""
			}
		})
	case "p2_gm":
		b.sm.Update(func(s *state.GameState) {
			if s.Player2Games > 0 {
				s.Player2Games--
				s.Winner = ""
			}
		})

	// ── SOZLAMALAR ─────────────────────────────
	case "set_p1":
		gs := b.sm.Get()
		b.sessions[chatID] = awaitP1Name
		b.api.Send(tgbotapi.NewMessage(chatID,
			fmt.Sprintf("1-o'yinchi ismini yozing:\n(hozir: %s)", gs.Player1Name)))
		return
	case "set_p2":
		gs := b.sm.Get()
		b.sessions[chatID] = awaitP2Name
		b.api.Send(tgbotapi.NewMessage(chatID,
			fmt.Sprintf("2-o'yinchi ismini yozing:\n(hozir: %s)", gs.Player2Name)))
		return
	case "set_race":
		gs := b.sm.Get()
		b.sessions[chatID] = awaitRaceTo
		b.api.Send(tgbotapi.NewMessage(chatID,
			fmt.Sprintf("Nechta ramka yutsa match g'olibi bo'ladi? (1-999)\n(hozir: %d)", gs.RaceTo)))
		return

	case "reset_frame":
		b.sm.ResetFrame()
	case "new_match":
		b.sm.NewMatch()
	case "noop":
		return
	}

	b.editPanel(chatID, msgID)
}

func (b *Bot) panelText() string {
	s := b.sm.Get()

	var statusLine string
	if s.Winner != "" {
		statusLine = fmt.Sprintf("\n🏆 G'OLIB: %s!", strings.ToUpper(s.Winner))
	} else {
		need1 := s.RaceTo - s.Player1Games
		need2 := s.RaceTo - s.Player2Games
		switch {
		case need1 < need2:
			statusLine = fmt.Sprintf("\n📊 %s yetaklamoqda (%d ramka kerak)", s.Player1Name, need1)
		case need2 < need1:
			statusLine = fmt.Sprintf("\n📊 %s yetaklamoqda (%d ramka kerak)", s.Player2Name, need2)
		default:
			statusLine = "\n⚖️ Teng!"
		}
	}

	return fmt.Sprintf(
		"🎱 BILLIARD SCOREBOARD\n"+
			"━━━━━━━━━━━━━━━━━━━━━━\n\n"+
			"🏅 MATCH HISOBI (ramkalar):\n"+
			"🔴 %-14s  %d ta\n"+
			"🔵 %-14s  %d ta\n\n"+
			"🎯 JORIY RAMKA (tosh):\n"+
			"🔴 %-14s  %d pts\n"+
			"🔵 %-14s  %d pts\n\n"+
			"🏁 Nechigacha: %d ramka%s",
		s.Player1Name, s.Player1Games,
		s.Player2Name, s.Player2Games,
		s.Player1Name, s.Player1FrameScore,
		s.Player2Name, s.Player2FrameScore,
		s.RaceTo, statusLine,
	)
}

func (b *Bot) panelKeyboard() tgbotapi.InlineKeyboardMarkup {
	s := b.sm.Get()

	p1FrameBtn := fmt.Sprintf("🔴 %s: %d tosh", s.Player1Name, s.Player1FrameScore)
	p2FrameBtn := fmt.Sprintf("🔵 %s: %d tosh", s.Player2Name, s.Player2FrameScore)

	p1WinsBtn := fmt.Sprintf("✅ %s RAMKA YUTDI!", s.Player1Name)
	p2WinsBtn := fmt.Sprintf("✅ %s RAMKA YUTDI!", s.Player2Name)

	p1UndoBtn := fmt.Sprintf("↩️ %s match -%d", s.Player1Name, 1)
	p2UndoBtn := fmt.Sprintf("↩️ %s match -%d", s.Player2Name, 1)

	return tgbotapi.NewInlineKeyboardMarkup(
		// Tosh hisobi — P1
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ -1", "p1_fm"),
			tgbotapi.NewInlineKeyboardButtonData(p1FrameBtn, "noop"),
			tgbotapi.NewInlineKeyboardButtonData("+1 ▶", "p1_fp"),
		),
		// Tosh hisobi — P2
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀ -1", "p2_fm"),
			tgbotapi.NewInlineKeyboardButtonData(p2FrameBtn, "noop"),
			tgbotapi.NewInlineKeyboardButtonData("+1 ▶", "p2_fp"),
		),
		// ── ASOSIY TUGMALAR: Ramka g'olibi ──
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(p1WinsBtn, "p1_wins"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(p2WinsBtn, "p2_wins"),
		),
		// Match hisobini tuzatish (xato bo'lsa)
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(p1UndoBtn, "p1_gm"),
			tgbotapi.NewInlineKeyboardButtonData(p2UndoBtn, "p2_gm"),
		),
		// Sozlamalar
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✏️ P1 ismi", "set_p1"),
			tgbotapi.NewInlineKeyboardButtonData("✏️ P2 ismi", "set_p2"),
			tgbotapi.NewInlineKeyboardButtonData(fmt.Sprintf("🏁 %d ta", s.RaceTo), "set_race"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Frame sifirla", "reset_frame"),
			tgbotapi.NewInlineKeyboardButtonData("🆕 Yangi match", "new_match"),
		),
	)
}

func (b *Bot) sendPanel(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, b.panelText())
	kb := b.panelKeyboard()
	msg.ReplyMarkup = kb
	b.api.Send(msg)
}

func (b *Bot) editPanel(chatID int64, msgID int) {
	kb := b.panelKeyboard()
	edit := tgbotapi.NewEditMessageText(chatID, msgID, b.panelText())
	edit.ReplyMarkup = &kb
	b.api.Send(edit)
}
