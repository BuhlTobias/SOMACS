package SOMACS

type Event[T any] struct {
	subscribers []*func(T)
}

func (ev *Event[T]) Subscribe(f *func(T)) {
	if ev.subscribers == nil {
		ev.subscribers = make([]*func(T), 0)
	}
	for i := range ev.subscribers {
		if ev.subscribers[i] == f {
			return
		}
	}
	ev.subscribers = append(ev.subscribers, f)
}

func (ev *Event[T]) Unsubscribe(f *func(T)) {
	for i := range ev.subscribers {
		if ev.subscribers[i] == f {
			ev.subscribers = append(ev.subscribers[:i], ev.subscribers[i+1:]...)
			return
		}
	}
}

func (ev *Event[T]) invoke(param T) bool {
	for i := range ev.subscribers {
		(*ev.subscribers[i])(param)
	}
	return ev.subscribers != nil && len(ev.subscribers) > 0
}
