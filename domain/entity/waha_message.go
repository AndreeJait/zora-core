package entity

// WAHAMessage is a lightweight representation of a WhatsApp message stored for reply chain resolution.
type WAHAMessage struct {
	ID          string `json:"id"`
	ChatID      string `json:"chat_id"`
	SenderPhone string `json:"sender_phone"`
	Body        string `json:"body"`
	IsReply     bool   `json:"is_reply"`
	QuotedMsgID string `json:"quoted_msg_id,omitempty"`
	FromMe      bool   `json:"from_me"` // true = outgoing (zora's message)
	Timestamp   int64  `json:"timestamp"`
}