package timing

import (
	"context"
	"testing"
	"time"
)

func TestWaiter(t *testing.T) {
	for _, testCase := range []func(pred *pollPred) (waiter <-chan struct{}){
		func(pred *pollPred) (waiter <-chan struct{}) {
			return CreateWaiterCh(
				func() { pred.Pred(context.Background()) },
				func() { pred.Pred(context.Background()) },
				func() { pred.Pred(context.Background()) },
				func() { pred.Pred(context.Background()) },
				func() { pred.Pred(context.Background()) },
			)
		},
		func(pred *pollPred) (waiter <-chan struct{}) {
			return CreateRepeatedWaiterCh(
				func() { pred.Pred(context.Background()) },
				5,
			)
		},
	} {
		pred := newPollPred()
		waiter := testCase(pred)
		unblocked := make(chan struct{})
		go func() {
			<-waiter
			close(unblocked)
		}()

		for i := 0; i < 5; i++ {
			select {
			case <-unblocked:
				t.Errorf("waiter must not return at this point")
			default:
			}
			pred.Unblock()
		}

		select {
		case <-unblocked:
		case <-time.After(time.Second):
			t.Errorf("waiter must return at this point")
		}
	}
}
