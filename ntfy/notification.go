package ntfy

type Notification struct {
	Topic   string `json:"topic"`
	Title   string `json:"title"`
	Message string `json:"message"`

	Priority int    `json:"priority,omitempty"`
	Attach   string `json:"attach,omitempty"`
	Filename string `json:"filename,omitempty"`
	Click    string `json:"click,omitempty"`

	Tags []string `json:"tags,omitempty"`
}

type NotificationOption func(*Notification)

func WithPriority(priority int) NotificationOption {
	return func(n *Notification) {
		n.Priority = priority
	}
}

func WithAttachment(attach string) NotificationOption {
	return func(n *Notification) {
		n.Attach = attach
	}
}

func WithFilename(filename string) NotificationOption {
	return func(n *Notification) {
		n.Filename = filename
	}
}

func WithClickURL(click string) NotificationOption {
	return func(n *Notification) {
		n.Click = click
	}
}

func WithTags(tags []string) NotificationOption {
	return func(n *Notification) {
		n.Tags = tags
	}

}

func NewNotification(topic, title, message string, opts ...NotificationOption) Notification {
	notification := Notification{
		Topic:   topic,
		Title:   title,
		Message: message,
	}

	for _, opt := range opts {
		opt(&notification)
	}

	return notification
}
