package entity

import (
	"strings"
	"time"
)

// WAHAWebhookEvent represents the top-level envelope received from WAHA webhook.
type WAHAWebhookEvent struct {
	Event   string             `json:"event"`
	Session string             `json:"session"`
	Me      WAHAContact        `json:"me"`
	Payload WAHAWebhookPayload `json:"payload"`
}

// WAHAContact represents a WhatsApp contact identity.
type WAHAContact struct {
	ID       string `json:"id"`
	PushName string `json:"pushName"`
}

// WAHAWebhookPayload represents the message payload inside a webhook event.
type WAHAWebhookPayload struct {
	ID           string              `json:"id"`
	From         string              `json:"from"`
	FromMe       bool                `json:"fromMe"`
	To           string              `json:"to"`
	Participant  string              `json:"participant"`
	Source       string              `json:"source"`
	AckName      string              `json:"ackName"`
	Body         string              `json:"body"`
	HasMedia     bool                `json:"hasMedia"`
	Timestamp    int64               `json:"timestamp"`
	MentionedIDs []string            `json:"mentionedIds"`
	Data         *WAHAMessageData    `json:"_data"`
	ReplyTo      *WAHAReplyTo        `json:"replyTo"`
}

// WAHAReplyTo represents the quoted/replied-to message metadata.
type WAHAReplyTo struct {
	ID          string `json:"id"`
	Participant string `json:"participant"`
	Body        string `json:"body"`
	HasMedia    bool   `json:"hasMedia"`
}

// WAHAMessageData represents the _data field from WAHA webhook payload.
type WAHAMessageData struct {
	ID     *WAHAMessageDataID `json:"id"`
	Author string             `json:"author"` // sender LID for group messages (e.g. "130605761167448@lid")
}

// WAHAMessageDataID represents _data.id — the canonical message identity.
type WAHAMessageDataID struct {
	Remote      string `json:"remote"`
	FromMe      bool   `json:"fromMe"`
	Participant string `json:"participant"`
	SenderPn    string `json:"senderPn"`
}

// WAHAIncomingMessage is the processed domain representation of an incoming WhatsApp message.
type WAHAIncomingMessage struct {
	Session      string
	FromMe       bool
	From         string   // best available sender identifier (phone@c.us or LID or group@g.us)
	ChatID       string   // chat ID to reply to (phone@c.us for private, group@g.us for group)
	MeID         string   // bot's own ID
	SenderPhone  string   // resolved phone number (without @c.us), may be empty in groups
	SenderLID    string   // raw LID (e.g. "130605761167448@lid"), may be empty
	Body         string
	MessageID    string
	Timestamp    time.Time
	HasMedia     bool
	MentionedIDs []string
	IsReply      bool
	QuotedMsgID  string
	QuotedBody   string
}

// IsGroupChat returns true if the chat ID indicates a WhatsApp group.
func IsGroupChat(chatID string) bool {
	return strings.HasSuffix(chatID, "@g.us")
}

// NewWAHAIncomingMessage converts a raw WAHAWebhookEvent into a domain WAHAIncomingMessage.
// Returns nil for events we don't process.
func NewWAHAIncomingMessage(event *WAHAWebhookEvent) *WAHAIncomingMessage {
	if event.Event == "message.any" {
		if event.Payload.Source == "api" {
			return nil
		}
	} else if event.Event == "message.ack" {
		if event.Payload.AckName != "DEVICE" {
			return nil
		}
		if event.Payload.Source == "api" {
			return nil
		}
	} else {
		return nil
	}

	chatID := ""
	dataParticipant := ""
	senderPn := ""
	dataAuthor := ""
	if event.Payload.Data != nil {
		if event.Payload.Data.ID != nil {
			chatID = event.Payload.Data.ID.Remote
			dataParticipant = event.Payload.Data.ID.Participant
			senderPn = event.Payload.Data.ID.SenderPn
		}
		dataAuthor = event.Payload.Data.Author
	}

	senderPhone := ""
	senderLID := ""

	from := event.Me.ID
	if !event.Payload.FromMe {
		// Resolve sender identity, preferring the most specific identifier:
		// senderPn (phone) > data.id.participant > data.author (LID) > payload.participant > payload.from
		if senderPn != "" && strings.HasSuffix(senderPn, "@c.us") {
			from = senderPn
			senderPhone = strings.TrimSuffix(senderPn, "@c.us")
		} else if dataParticipant != "" && dataParticipant != "out" && !strings.HasPrefix(dataParticipant, "out@") {
			from = dataParticipant
			if strings.HasSuffix(dataParticipant, "@lid") {
				senderLID = dataParticipant
			} else if strings.HasSuffix(dataParticipant, "@c.us") {
				senderPhone = strings.TrimSuffix(dataParticipant, "@c.us")
			}
		} else if dataAuthor != "" {
			// _data.author contains the sender's LID (e.g. "103332819558457:59@lid")
			senderLID = dataAuthor
			from = dataAuthor
		} else if event.Payload.Participant != "" && event.Payload.Participant != "out@c.us" && !strings.HasPrefix(event.Payload.Participant, "out@") {
			from = event.Payload.Participant
			if strings.HasSuffix(event.Payload.Participant, "@lid") {
				senderLID = event.Payload.Participant
			} else if strings.HasSuffix(event.Payload.Participant, "@c.us") {
				senderPhone = strings.TrimSuffix(event.Payload.Participant, "@c.us")
			}
		} else {
			from = event.Payload.From
			if strings.HasSuffix(event.Payload.From, "@c.us") {
				senderPhone = strings.TrimSuffix(event.Payload.From, "@c.us")
			}
		}
		if chatID == "" {
			chatID = event.Payload.From
		}
	} else {
		// fromMe: message sent from the bot's own account (e.g. user texting themselves).
		// Still extract the sender phone from payload.from and LID from _data.author.
		if strings.HasSuffix(event.Payload.From, "@c.us") {
			senderPhone = strings.TrimSuffix(event.Payload.From, "@c.us")
		}
		if dataAuthor != "" {
			senderLID = dataAuthor
		}
		if chatID == "" {
			chatID = event.Payload.To
		}
	}

	isReply := event.Payload.ReplyTo != nil
	var quotedMsgID, quotedBody string
	if isReply {
		quotedMsgID = event.Payload.ReplyTo.ID
		quotedBody = event.Payload.ReplyTo.Body
	}

	return &WAHAIncomingMessage{
		Session:      event.Session,
		FromMe:       event.Payload.FromMe,
		From:         from,
		ChatID:       chatID,
		MeID:         event.Me.ID,
		SenderPhone:  senderPhone,
		SenderLID:    senderLID,
		Body:         event.Payload.Body,
		MessageID:    event.Payload.ID,
		Timestamp:    time.UnixMilli(event.Payload.Timestamp),
		HasMedia:     event.Payload.HasMedia,
		MentionedIDs: event.Payload.MentionedIDs,
		IsReply:      isReply,
		QuotedMsgID:  quotedMsgID,
		QuotedBody:   quotedBody,
	}
}
