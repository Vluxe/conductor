package conductor

type MessageOpCode int

const (
	Bind MessageOpCode = 1 << iota
	Unbind
	Write
	Info
)

// Struct of message json.
type Message struct {
	Token       string        `json:"token"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	ChannelName string        `json:"channel_name"`
	OpCode      MessageOpCode `json:"opcode"`
}
