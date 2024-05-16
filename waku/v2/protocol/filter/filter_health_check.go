package filter

import (
	"context"
	"time"
)

func (wf *WakuFilterLightNode) PingPeers() {
	//Send a ping to all the peers and report their status to corresponding subscriptions
	// Alive or not or set state of subcription??
	for _, peer := range wf.subscriptions.GetSubscribedPeers() {
		err := wf.Ping(context.TODO(), peer)
		if err != nil {
			subscriptions := wf.subscriptions.GetAllSubscriptionsForPeer(peer)
			for _, subscription := range subscriptions {
				//Indicating that subscription is closing
				//This feels like a hack, but taking this approach for now so as to avoid refactoring.
				subscription.Closing <- true
			}
		}
	}
}

func (wf *WakuFilterLightNode) FilterHealthCheckLoop() {
	wf.CommonService.WaitGroup().Add(1)
	defer wf.WaitGroup().Done()
	for {
		//TODO: Do we have to wait for wf.ctx context completion and exit as well?
		time.Sleep(wf.peerPingInterval)
		wf.PingPeers()
	}
}
