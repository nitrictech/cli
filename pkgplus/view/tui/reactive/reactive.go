package reactive

import (
	tea "github.com/charmbracelet/bubbletea"
)

// TODO: simplify this to helpers that allow for reacting to channels and method that helps aggregate channels or convert subscriptions into channel. Make more stateless.

type Subscription struct {
	aggregateChannel chan tea.Msg
}

type SubscriberMethod[T any] func(fn func(T))

func pipeSubscriptionToChannel[T any](sub SubscriberMethod[T]) chan any {
	channel := make(chan any)

	sub(func(t T) {
		channel <- t
	})

	return channel
}

type Message struct {
	Msg tea.Msg
}

func ReceiveOn[T any](sub *Subscription, channel chan T) {
	go func() {
		for {
			sub.aggregateChannel <- <-channel
		}
	}()
}

func ListenFor[T any](sub *Subscription, in SubscriberMethod[T]) {
	go func() {
		channel := pipeSubscriptionToChannel(in)
		for {
			sub.aggregateChannel <- <-channel
		}
	}()
}

func ListManyFor[T any](sub *Subscription, in ...SubscriberMethod[T]) tea.Cmd {
	for _, s := range in {
		ListenFor(sub, s)
	}

	return sub.AwaitNextMsg()
}

func (r *Subscription) AwaitNextMsg() tea.Cmd {
	return func() tea.Msg {
		msg := <-r.aggregateChannel
		return Message{
			Msg: msg,
		}
	}
}

func NewSubscriber() *Subscription {
	return &Subscription{
		aggregateChannel: make(chan tea.Msg),
	}
}

type ChanMsg[T any] struct {
	Source <-chan T
	Ok     bool
	Value  T
}

func AwaitChannel[T any](channel <-chan T) tea.Cmd {
	return func() tea.Msg {

		msg, ok := <-channel

		return ChanMsg[T]{
			Source: channel,
			Ok:     ok,
			Value:  msg,
		}
	}
}
