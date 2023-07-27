package rendezvous

import (
	"context"
	"math/rand"
	"sort"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)

type RendezvousPointIterator struct {
	rendezvousPoints []*RendezvousPoint
}

func NewRendezvousPointIterator(rendezvousPoints []peer.ID) *RendezvousPointIterator {
	var rendevousPoints []*RendezvousPoint
	for _, rp := range rendezvousPoints {
		rendevousPoints = append(rendevousPoints, NewRendezvousPoint(rp))
	}

	return &RendezvousPointIterator{
		rendezvousPoints: rendevousPoints,
	}
}

func (r *RendezvousPointIterator) RendezvousPoints() []*RendezvousPoint {
	return r.rendezvousPoints
}

func (r *RendezvousPointIterator) Next(ctx context.Context) <-chan *RendezvousPoint {
	var dialableRP []*RendezvousPoint
	now := time.Now()
	for _, rp := range r.rendezvousPoints {
		if now.After(rp.NextTry()) {
			dialableRP = append(dialableRP, rp)
		}
	}

	result := make(chan *RendezvousPoint, 1)

	if len(dialableRP) > 0 {
		result <- r.rendezvousPoints[rand.Intn(len(r.rendezvousPoints))] // nolint: gosec
	} else {
		if len(r.rendezvousPoints) > 0 {
			sort.Slice(r.rendezvousPoints, func(i, j int) bool {
				return r.rendezvousPoints[i].nextTry.Before(r.rendezvousPoints[j].nextTry)
			})

			tryIn := r.rendezvousPoints[0].NextTry().Sub(now)
			timer := time.NewTimer(tryIn)
			defer timer.Stop()

			select {
			case <-ctx.Done():
				break
			case <-timer.C:
				result <- r.rendezvousPoints[0]
			}
		}
	}

	close(result)
	return result
}
