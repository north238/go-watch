package notifier

type NopNotifier struct{}

func (nop *NopNotifier) Notify(message string) error {
	return nil
}
