package xactor

import (
	"context"
	"fmt"
	"sync"
)

var (
	mu     sync.RWMutex
	actors map[string]*ActorGroutine
)

func init() {
	actors = make(map[string]*ActorGroutine)
}

func registerActor(actor *ActorGroutine) error {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := actors[actor.state.Name()]; ok {
		return fmt.Errorf("actor %v is repeated", actor.state.Name())
	}
	actors[actor.state.Name()] = actor
	return nil
}

func deregisterActor(actor *ActorGroutine) {
	mu.Lock()
	defer mu.Unlock()
	delete(actors, actor.state.Name())
}

func GetActor(name string) (*ActorGroutine, error) {
	mu.RLock()
	actor := actors[name]
	mu.RUnlock()
	if actor == nil {
		return nil, fmt.Errorf("actor[%v] is nil", name)
	}
	return actor, nil
}

func CloseAll(ctx context.Context) {
	as := make([]*ActorGroutine, 0)

	mu.Lock()
	for _, actor := range actors {
		as = append(as, actor)
	}
	mu.Unlock()

	for _, actor := range as {
		actor.Close(ctx)
	}
}
