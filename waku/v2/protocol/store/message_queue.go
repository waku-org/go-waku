package store

type MessageQueue struct {
	messages    []IndexedWakuMessage
	maxMessages int
}

func (self *MessageQueue) Push(msg IndexedWakuMessage) {
	self.messages = append(self.messages, msg)

	if self.maxMessages != 0 && len(self.messages) > self.maxMessages {
		numToPop := len(self.messages) - self.maxMessages
		self.messages = self.messages[numToPop:len(self.messages)]
	}
}

func (self *MessageQueue) Messages() []IndexedWakuMessage {
	return self.messages
}

func NewMessageQueue(maxMessages int) *MessageQueue {
	return &MessageQueue{
		maxMessages: maxMessages,
	}
}
