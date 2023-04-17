package subscription

type channelKey struct {
	id      string
	channel string
}

func newChannelKey(id string, channel string) channelKey {
	return channelKey{
		id:      id,
		channel: channel,
	}
}
