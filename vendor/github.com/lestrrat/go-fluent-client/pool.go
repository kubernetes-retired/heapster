package fluent

import "sync"

var msgpool = sync.Pool{
	New: allocMessage,
}

func allocMessage() interface{} {
	return &Message{}
}

func getMessage() *Message {
	return msgpool.Get().(*Message)
}

func releaseMessage(m *Message) {
	m.clear()
	msgpool.Put(m)
}
