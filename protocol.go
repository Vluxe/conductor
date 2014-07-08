package conductor

type Message struct {
	Token       string `json:"token"`
	Body        string `json:"body"`
	ChannelName string `json:"channel_name"`
}
