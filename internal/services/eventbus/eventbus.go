package eventbus

import (
	"context"
	"fmt"
	"log/slog"
	"subpub/pkg/subpub"
	"sync"
)

type Eventbus struct {
	log *slog.Logger
	sp  subpub.SubPub
}

type loggedSubscription struct {
	sub     subpub.Subscription
	log     *slog.Logger
	subject string
	once    sync.Once
}

func (l *loggedSubscription) Unsubscribe() {
	const op = "eventbus.Unsubscribe"
	l.once.Do(func() {
		log := l.log.With(
			slog.String("op", op),
			slog.String("subject", l.subject),
		)
		log.Info("starting unsubscribe")
		l.sub.Unsubscribe()
		log.Info("unsubscribe completed")
	})
}

func (e *Eventbus) Subscribe(subject string, cb subpub.MessageHandler) (subpub.Subscription, error) {
	const op = "eventbus.Subscribe"
	if subject == "" {
		return nil, fmt.Errorf("%s: empty subject", op)
	}
	if cb == nil {
		return nil, fmt.Errorf("%s: nil callback", op)
	}

	log := e.log.With(
		slog.String("op", op),
		slog.String("subject", subject),
	)

	sub, err := e.sp.Subscribe(subject, cb)
	if err != nil {
		log.Error("subscribe failed", "error", err)
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	log.Info("subscribe created")
	return &loggedSubscription{
		sub:     sub,
		log:     e.log,
		subject: subject,
	}, nil
}

func (e *Eventbus) Publish(subject string, msg interface{}) error {
	const op = "eventbus.Publish"
	if subject == "" {
		return fmt.Errorf("%s: empty subject", op)
	}

	log := e.log.With(
		slog.String("op", op),
		slog.String("subject", subject),
	)

	err := e.sp.Publish(subject, msg)
	if err != nil {
		log.Error("publish failed", "error", err)
		return fmt.Errorf("%s: %w", op, err)
	}

	log.Debug("message published")
	return nil
}

func (e *Eventbus) Close(ctx context.Context) error {
	const op = "eventbus.Close"
	log := e.log.With(slog.String("op", op))

	log.Info("starting shutdown")
	defer log.Info("shutdown completed")

	if err := e.sp.Close(ctx); err != nil {
		log.Error("shutdown failed", "error", err)
		return fmt.Errorf("%s: %w", op, err)
	}
	return nil
}

func NewEventBus(log *slog.Logger, sp subpub.SubPub) (*Eventbus, error) {
	if log == nil {
		return nil, fmt.Errorf("nil logger")
	}
	if sp == nil {
		return nil, fmt.Errorf("nil subpub")
	}
	return &Eventbus{
		log: log,
		sp:  sp,
	}, nil
}
