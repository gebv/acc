package ffsm

type AbstractMessageOfChannel interface{}

type Channel chan AbstractMessageOfChannel

func (c Channel) Close() (closed bool) {
	defer func() {
		if recover() != nil {
			closed = false
		}
	}()

	close(c)
	return true
}

func (c Channel) Send(msg AbstractMessageOfChannel) (failed bool) {
	defer func() {
		if recover() != nil {
			failed = false
		}
	}()

	select {
	case c <- msg:
	default:
		return false
	}

	return true
}
